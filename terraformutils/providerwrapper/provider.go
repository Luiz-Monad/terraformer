// Copyright 2018 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package providerwrapper //nolint

import (
	"context"
	encjson "encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils/terraformerstring"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/configschema"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/json"
	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// DefaultDataDir is the default directory for storing local data.
const DefaultDataDir = ".terraform"

// DefaultPluginVendorDir is the location in the config directory to look for
// user-added plugin binaries. Terraform only reads from this path if it
// exists, it is never created by terraform.
const DefaultPluginVendorDirV12 = "terraform.d/plugins/" + pluginMachineName

// pluginMachineName is the directory name used in new plugin paths.
const pluginMachineName = runtime.GOOS + "_" + runtime.GOARCH

type ProviderWrapper struct {
	context      context.Context
	provider     tfprotov5.ProviderServer
	client       *plugin.Client
	rpcClient    plugin.ClientProtocol
	providerName string
	config       cty.Value
	schema       *tfprotov5.GetProviderSchemaResponse
	retryCount   int
	retrySleepMs int
}

func NewProviderWrapper(providerName string, providerConfig cty.Value, verbose bool, options ...map[string]int) (*ProviderWrapper, error) {
	p := &ProviderWrapper{retryCount: 5, retrySleepMs: 300}
	p.providerName = providerName
	p.config = providerConfig

	if len(options) > 0 {
		retryCount, hasOption := options[0]["retryCount"]
		if hasOption {
			p.retryCount = retryCount
		}
		retrySleepMs, hasOption := options[0]["retrySleepMs"]
		if hasOption {
			p.retrySleepMs = retrySleepMs
		}
	}

	err := p.initProvider(verbose)

	return p, err
}

func (p *ProviderWrapper) Kill() {
	p.client.Kill()
}

func (p *ProviderWrapper) GetSchema() (*tfprotov5.GetProviderSchemaResponse, error) {
	if p.schema == nil {
		r, err := p.provider.GetProviderSchema(p.context, &tfprotov5.GetProviderSchemaRequest{})
		if err != nil {
			return nil, err
		}
		p.schema = r
	}
	return p.schema, nil
}

func (p *ProviderWrapper) GetReadOnlyAttributes(resourceTypes []string) (map[string][]string, error) {
	r, err := p.GetSchema()

	if err != nil {
		return nil, err
	}
	readOnlyAttributes := map[string][]string{}
	for resourceName, obj := range r.ResourceSchemas {
		if terraformerstring.ContainsString(resourceTypes, resourceName) {
			readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^id$")
			for _, v := range obj.Block.Attributes {
				if !v.Optional && !v.Required {
					if v.Type.Is(tftypes.List{}) || v.Type.Is(tftypes.Set{}) {
						readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^"+v.Name+"\\.(.*)")
					} else {
						readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^"+v.Name+"$")
					}
				}
			}
			readOnlyAttributes[resourceName] = p.readObjBlocks(obj.Block.BlockTypes, readOnlyAttributes[resourceName], "-1")
		}
	}
	return readOnlyAttributes, nil
}

func (p *ProviderWrapper) readObjBlocks(block []*tfprotov5.SchemaNestedBlock, readOnlyAttributes []string, parent string) []string {
	for _, v := range block {
		k := v.TypeName
		if len(v.Block.BlockTypes) > 0 {
			if parent == "-1" {
				readOnlyAttributes = p.readObjBlocks(v.Block.BlockTypes, readOnlyAttributes, k)
			} else {
				readOnlyAttributes = p.readObjBlocks(v.Block.BlockTypes, readOnlyAttributes, parent+"\\.[0-9]+\\."+k)
			}
		}
		fieldCount := 0
		for _, l := range v.Block.Attributes {
			key := l.Name
			if !l.Optional && !l.Required {
				fieldCount++
				switch v.Nesting {
				case tfprotov5.SchemaNestedBlockNestingModeList:
					if parent == "-1" {
						readOnlyAttributes = append(readOnlyAttributes, "^"+k+"\\.[0-9]+\\."+key+"($|\\.[0-9]+|\\.#)")
					} else {
						readOnlyAttributes = append(readOnlyAttributes, "^"+parent+"\\.(.*)\\."+key+"$")
					}
				case tfprotov5.SchemaNestedBlockNestingModeSet:
					if parent == "-1" {
						readOnlyAttributes = append(readOnlyAttributes, "^"+k+"\\.[0-9]+\\."+key+"$")
					} else {
						readOnlyAttributes = append(readOnlyAttributes, "^"+parent+"\\.(.*)\\."+key+"($|\\.(.*))")
					}
				case tfprotov5.SchemaNestedBlockNestingModeMap:
					readOnlyAttributes = append(readOnlyAttributes, parent+"\\."+key)
				default:
					readOnlyAttributes = append(readOnlyAttributes, parent+"\\."+key+"$")
				}
			}
		}
		if fieldCount == len(v.Block.Attributes) && fieldCount > 0 && len(v.Block.BlockTypes) == 0 {
			readOnlyAttributes = append(readOnlyAttributes, "^"+k)
		}
	}
	return readOnlyAttributes
}

func (p *ProviderWrapper) Refresh(info *terraform.InstanceInfo, state *terraform.InstanceState) (*terraform.InstanceState, error) {
	provSchema, err := p.GetSchema()
	if err != nil {
		return nil, err
	}
	impliedType := configschema.WrapBlock(provSchema.ResourceSchemas[info.Type].Block).ImpliedType()
	priorState, err := state.AttrsAsObjectValue(impliedType)
	if err != nil {
		return nil, err
	}
	successReadResource := false
	var resp *tfprotov5.ReadResourceResponse
	for i := 0; i < p.retryCount; i++ {
		resp, err = p.provider.ReadResource(p.context, &tfprotov5.ReadResourceRequest{
			TypeName:     info.Type,
			CurrentState: NewDynamicValue(priorState),
			Private:      []byte{},
		})
		if err != nil {
			log.Println(err)
			log.Println(resp.Diagnostics)
			log.Printf("WARN: Fail read resource from provider for resource %s, wait %dms before retry\n", info.Id, p.retrySleepMs)
			time.Sleep(time.Duration(p.retrySleepMs) * time.Millisecond)
			continue
		} else {
			if resp.NewState == nil {
				log.Printf("WARN: Read resource response is null for resource %s, wait %dms before retry\n", info.Id, p.retrySleepMs)
				time.Sleep(time.Duration(p.retrySleepMs) * time.Millisecond)
				continue
			}
			successReadResource = true
			break
		}
	}

	var newState *tfprotov5.DynamicValue
	if !successReadResource {
		log.Println("Fail read resource from provider, trying import command")
		// retry with regular import command - without resource attributes
		importResponse, err := p.provider.ImportResourceState(p.context, &tfprotov5.ImportResourceStateRequest{
			TypeName: info.Type,
			ID:       state.ID,
		})
		if err != nil {
			return nil, err
		}
		if len(importResponse.ImportedResources) == 0 {
			return nil, errors.New("not able to import resource for a given ID")
		}
		newState = importResponse.ImportedResources[0].State
	} else {
		if resp.NewState == nil {
			msg := fmt.Sprintf("ERROR: Read resource response is null for resource %s", info.Id)
			return nil, errors.New(msg)
		}
		newState = resp.NewState
	}

	newStateVal, err := UnmarshallDynamicValue(newState, impliedType)
	if err != nil {
		return nil, err
	}
	return terraform.NewInstanceStateShimmedFromValue(newStateVal, int(provSchema.ResourceSchemas[info.Type].Version)), nil
}

func (p *ProviderWrapper) initProvider(verbose bool) error {
	reattach := getReattachProviders()
	providerFilePath, err := getProviderFileName(p.providerName)
	if err != nil && reattach == nil {
		return err
	}
	options := hclog.LoggerOptions{
		Name:   "plugin",
		Level:  hclog.Error,
		Output: os.Stdout,
	}
	if verbose {
		options.Level = hclog.Trace
	}
	logger := hclog.New(&options)
	cmd := exec.Command(providerFilePath)
	var unversionedPlugins plugin.PluginSet
	if reattach != nil {
		cmd = nil
		// github.com/hashicorp/terraform@v1.4.5/internal/command/meta_providers.go/unmanagedProviderFactory
		unversionedPlugins = tfplugin.VersionedPlugins[reattach.ProtocolVersion]
	}
	p.client = plugin.NewClient(&plugin.ClientConfig{
		Cmd:              cmd,
		Reattach:         reattach,
		HandshakeConfig:  tfplugin.Handshake,
		VersionedPlugins: tfplugin.VersionedPlugins,
		Plugins:          unversionedPlugins,
		Managed:          (reattach == nil),
		Logger:           logger,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		AutoMTLS:         true,
	})
	p.rpcClient, err = p.client.Client()
	if err != nil {
		return err
	}
	raw, err := p.rpcClient.Dispense(tfplugin.ProviderPluginName)
	if err != nil {
		return err
	}

	p.provider = raw.(tfprotov5.ProviderServer)
	p.context = raw.(tfplugin.ClientContext).Context()

	schema, err := p.GetSchema()
	if err != nil {
		return err
	}
	if p.config.IsNull() {
		p.config = cty.EmptyObjectVal
	}
	config, err := configschema.WrapBlock(schema.Provider.Block).CoerceValue(p.config)
	if err != nil {
		return err
	}
	_, err = p.provider.ConfigureProvider(p.context, &tfprotov5.ConfigureProviderRequest{
		TerraformVersion: "v1.0.0",
		Config:           NewDynamicValue(config),
	})
	if err != nil {
		return err
	}

	return nil
}

func getProviderFileName(providerName string) (string, error) {
	defaultDataDir := os.Getenv("TF_DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = DefaultDataDir
	}
	providerFilePath, err := getProviderFileNameV13andV14(defaultDataDir, providerName)
	if err != nil || providerFilePath == "" {
		providerFilePath, err = getProviderFileNameV13andV14(os.Getenv("HOME")+string(os.PathSeparator)+
			".terraform.d", providerName)
	}
	if err != nil || providerFilePath == "" {
		return getProviderFileNameV12(providerName)
	}
	return providerFilePath, nil
}

func getProviderFileNameV13andV14(prefix, providerName string) (string, error) {
	// Read terraform v14 file path
	registryDir := prefix + string(os.PathSeparator) + "providers" + string(os.PathSeparator) +
		"registry.terraform.io"
	providerDirs, err := ioutil.ReadDir(registryDir)
	if err != nil {
		// Read terraform v13 file path
		registryDir = prefix + string(os.PathSeparator) + "plugins" + string(os.PathSeparator) +
			"registry.terraform.io"
		providerDirs, err = ioutil.ReadDir(registryDir)
		if err != nil {
			return "", err
		}
	}
	providerFilePath := ""
	for _, providerDir := range providerDirs {
		pluginPath := registryDir + string(os.PathSeparator) + providerDir.Name() +
			string(os.PathSeparator) + providerName
		dirs, err := ioutil.ReadDir(pluginPath)
		if err != nil {
			continue
		}
		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			for _, dir := range dirs {
				fullPluginPath := pluginPath + string(os.PathSeparator) + dir.Name() +
					string(os.PathSeparator) + runtime.GOOS + "_" + runtime.GOARCH
				files, err := ioutil.ReadDir(fullPluginPath)
				if err == nil {
					for _, file := range files {
						if strings.HasPrefix(file.Name(), "terraform-provider-"+providerName) {
							providerFilePath = fullPluginPath + string(os.PathSeparator) + file.Name()
						}
					}
				}
			}
		}
	}
	return providerFilePath, nil
}

func getProviderFileNameV12(providerName string) (string, error) {
	defaultDataDir := os.Getenv("TF_DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = DefaultDataDir
	}
	pluginPath := defaultDataDir + string(os.PathSeparator) + "plugins" + string(os.PathSeparator) + runtime.GOOS + "_" + runtime.GOARCH
	files, err := ioutil.ReadDir(pluginPath)
	if err != nil {
		pluginPath = os.Getenv("HOME") + string(os.PathSeparator) + "." + DefaultPluginVendorDirV12
		files, err = ioutil.ReadDir(pluginPath)
		if err != nil {
			return "", err
		}
	}
	providerFilePath := ""
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasPrefix(file.Name(), "terraform-provider-"+providerName) {
			providerFilePath = pluginPath + string(os.PathSeparator) + file.Name()
		}
	}
	return providerFilePath, nil
}

func getReattachProviders() *plugin.ReattachConfig {
	// copied from github.com/hashicorp/terraform@v1.4.5/main.go/parseReattachProviders
	reattach := os.Getenv("TF_REATTACH_PROVIDERS")
	if reattach == "" {
		return nil
	}
	type reattachConfig struct {
		Protocol        string
		ProtocolVersion int
		Addr            struct {
			Network string
			String  string
		}
		Pid  int
		Test bool
	}
	var m map[string]reattachConfig
	err := encjson.Unmarshal([]byte(reattach), &m)
	if err != nil {
		log.Println("Invalid format for TF_REATTACH_PROVIDERS: %w", err)
	}
	for p, c := range m {
		var addr net.Addr
		switch c.Addr.Network {
		case "unix":
			addr, err = net.ResolveUnixAddr("unix", c.Addr.String)
			if err != nil {
				log.Printf("Invalid unix socket path %q for %q: %v", c.Addr.String, p, err)
			}
		case "tcp":
			addr, err = net.ResolveTCPAddr("tcp", c.Addr.String)
			if err != nil {
				log.Printf("Invalid TCP address %q for %q: %v", c.Addr.String, p, err)
			}
		default:
			log.Printf("Unknown address type %q for %q", c.Addr.Network, p)
		}
		// we only care about the first one
		return &plugin.ReattachConfig{
			Protocol:        plugin.Protocol(c.Protocol),
			ProtocolVersion: c.ProtocolVersion,
			Pid:             c.Pid,
			Test:            c.Test,
			Addr:            addr,
		}
	}
	return nil
}

func GetProviderVersion(providerName string) string {
	providerFilePath, err := getProviderFileName(providerName)
	if err != nil {
		log.Println("Can't find provider file path. Ensure that you are following https://www.terraform.io/docs/configuration/providers.html#third-party-plugins.")
		return ""
	}
	t := strings.Split(providerFilePath, string(os.PathSeparator))
	providerFileName := t[len(t)-1]
	providerFileNameParts := strings.Split(providerFileName, "_")
	if len(providerFileNameParts) < 2 {
		log.Println("Can't find provider version. Ensure that you are following https://www.terraform.io/docs/configuration/providers.html#plugin-names-and-versions.")
		return ""
	}
	providerVersion := providerFileNameParts[1]
	return "~> " + strings.TrimPrefix(providerVersion, "v")
}

func NewDynamicValue(val cty.Value) *tfprotov5.DynamicValue {
	mp, err := msgpack.Marshal(val, val.Type())
	if err != nil {
		panic(val)
	}
	return &tfprotov5.DynamicValue{
		MsgPack: mp,
	}
}

func UnmarshallDynamicValue(val *tfprotov5.DynamicValue, ty cty.Type) (cty.Value, error) {
	if val == nil {
		return cty.NullVal(ty), nil
	}
	switch {
	case len(val.MsgPack) > 0:
		return msgpack.Unmarshal(val.MsgPack, ty)
	case len(val.JSON) > 0:
		return json.Unmarshal(val.JSON, ty)
	}
	return cty.NullVal(ty), nil
}
