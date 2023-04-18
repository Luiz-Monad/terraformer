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

package terraformutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils/providerwrapper"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type BaseResource struct {
	Tags map[string]string `json:"tags,omitempty"`
}

func NewTfState(resources []Resource) *terraform.State {
	tfstate := &terraform.State{
		Version:   3, //internal/legacy/terraform/state.go
		TFVersion: "v1.0.0",
		Serial:    1,
	}
	outputs := map[string]*terraform.OutputState{}
	for _, r := range resources {
		for k, v := range r.Outputs {
			outputs[k] = v
		}
	}
	tfstate.Modules = []*terraform.ModuleState{
		{
			Path:      []string{"root"},
			Resources: map[string]*terraform.ResourceState{},
			Outputs:   outputs,
		},
	}
	for _, resource := range resources {
		resourceState := &terraform.ResourceState{
			Type:     resource.InstanceInfo.Type,
			Primary:  resource.InstanceState,
			Provider: "provider." + resource.Provider,
		}
		tfstate.Modules[0].Resources[resource.InstanceInfo.Type+"."+resource.ResourceName] = resourceState
	}
	return tfstate
}

func PrintTfState(resources []Resource) ([]byte, error) {
	state := NewTfState(resources)
	var buf bytes.Buffer
	err := writeState(state, &buf)
	return buf.Bytes(), err
}

func RefreshResources(resources []*Resource, provider *providerwrapper.ProviderWrapper, slowProcessingResources [][]*Resource) ([]*Resource, error) {

	DoWorkPooled(resources, 16, func(resource *Resource) (**Resource, error) {
		RefreshResource(resource, provider)
		return nil, nil //continue regardless
	})

	DoWorkPooled(slowProcessingResources, len(slowProcessingResources), func(resources []*Resource) (*[]*Resource, error) {
		for _, resource := range resources {
			RefreshResource(resource, provider)
		}
		return nil, nil //continue regardless
	})

	refreshedResources := []*Resource{}

	for _, r := range resources {
		if r.InstanceState != nil && r.InstanceState.ID != "" {
			refreshedResources = append(refreshedResources, r)
		} else {
			log.Printf("ERROR: Unable to refresh resource %s", r.ResourceName)
		}
	}

	for _, resourceGroup := range slowProcessingResources {
		for _, r := range resourceGroup {
			if r.InstanceState != nil && r.InstanceState.ID != "" {
				refreshedResources = append(refreshedResources, r)
			} else {
				log.Printf("ERROR: Unable to refresh resource %s", r.ResourceName)
			}
		}
	}
	return refreshedResources, nil
}

func RefreshResourcesByProvider(providersMapping *ProvidersMapping, providerWrapper *providerwrapper.ProviderWrapper) error {
	allResources := providersMapping.ShuffleResources()
	slowProcessingResources := make(map[ProviderGenerator][]*Resource)
	regularResources := []*Resource{}
	for i := range allResources {
		resource := allResources[i]
		if resource.SlowQueryRequired {
			provider := providersMapping.MatchProvider(resource)
			if slowProcessingResources[provider] == nil {
				slowProcessingResources[provider] = []*Resource{}
			}
			slowProcessingResources[provider] = append(slowProcessingResources[provider], resource)
		} else {
			regularResources = append(regularResources, resource)
		}
	}

	var spResourcesList [][]*Resource
	for p := range slowProcessingResources {
		spResourcesList = append(spResourcesList, slowProcessingResources[p])
	}

	refreshedResources, err := RefreshResources(regularResources, providerWrapper, spResourcesList)
	if err != nil {
		return err
	}

	providersMapping.SetResources(refreshedResources)
	return nil
}

func RefreshResource(r *Resource, provider *providerwrapper.ProviderWrapper) {
	log.Println("Refreshing state...", r.InstanceInfo.Id)
	r.Refresh(provider)
}

func IgnoreKeys(resourcesTypes []string, p *providerwrapper.ProviderWrapper) map[string][]string {
	readOnlyAttributes, err := p.GetReadOnlyAttributes(resourcesTypes)
	if err != nil {
		log.Println("plugin error 2:", err)
		return map[string][]string{}
	}
	return readOnlyAttributes
}

func ParseFilterValues(value string) []string {
	var values []string

	valueBuffering := true
	wrapped := false
	var valueBuffer []byte
	for i := 0; i < len(value); i++ {
		if value[i] == '\'' {
			wrapped = !wrapped
			continue
		} else if value[i] == ':' {
			if len(valueBuffer) == 0 {
				continue
			} else if valueBuffering && !wrapped {
				values = append(values, string(valueBuffer))
				valueBuffering = false
				valueBuffer = []byte{}
				continue
			}
		}
		valueBuffering = true
		valueBuffer = append(valueBuffer, value[i])
	}
	if len(valueBuffer) > 0 {
		values = append(values, string(valueBuffer))
	}

	return values
}

func FilterCleanup(s *Service, isInitial bool) {
	if len(s.Filter) == 0 {
		return
	}
	var newListOfResources []Resource
	for _, resource := range s.Resources {
		allPredicatesTrue := true
		for _, filter := range s.Filter {
			if filter.isInitial() == isInitial {
				allPredicatesTrue = allPredicatesTrue && filter.Filter(resource)
			}
		}
		if allPredicatesTrue && !ContainsResource(newListOfResources, resource) {
			newListOfResources = append(newListOfResources, resource)
		}
	}
	s.Resources = newListOfResources
}

func ContainsResource(s []Resource, e Resource) bool {
	for _, a := range s {
		if a.InstanceInfo.Id == e.InstanceInfo.Id {
			return true
		}
	}
	return false
}

// WriteState writes a state somewhere in a binary format.
// from internal\legacy\terraform\state.go
func writeState(d *terraform.State, dst io.Writer) error {
	// writing a nil state is a noop.
	if d == nil {
		return nil
	}

	// make sure we have no uninitialized fields
	// d.init()

	// Make sure it is sorted
	// d.sort()

	// Ensure the version is set
	d.Version = 3 //internal/legacy/terraform/state.go

	// If the TFVersion is set, verify it. We used to just set the version
	// here, but this isn't safe since it changes the MD5 sum on some remote
	// state storage backends such as Atlas. We now leave it be if needed.
	if d.TFVersion != "" {
		d.TFVersion = "v1.0.0"
		// if _, err := version.NewVersion(d.TFVersion); err != nil {
		// 	return fmt.Errorf(
		// 		"Error writing state, invalid version: %s\n\n"+
		// 			"The Terraform version when writing the state must be a semantic\n"+
		// 			"version.",
		// 		d.TFVersion)
		// }
	}

	// Encode the data in a human-friendly way
	data, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		return fmt.Errorf("Failed to encode state: %s", err)
	}

	// We append a newline to the data because MarshalIndent doesn't
	data = append(data, '\n')

	// Write the data out to the dst
	if _, err := io.Copy(dst, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("Failed to write state: %v", err)
	}

	return nil
}

func DoWorkPooled[T any](items []T, poolSize int, task func(T) (*T, error)) ([]T, error) {
	if poolSize == 0 { // 0 means sequential
		output := []T{}
		for _, item := range items {
			if out, err := task(item); err != nil {
				return output, err
			} else {
				if out != nil {
					output = append(output, *out)
				}
			}
		}
		return output, nil
	}

	numOfItems := len(items)
	input := make(chan T, numOfItems)
	output := make(chan T, numOfItems)
	errout := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(numOfItems)
	for _, item := range items {
		input <- item
	}
	close(input)

	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < poolSize; i++ {
		go func() {
			for {
				select {
				case item, ok := <-input:
					if !ok {
						// Input is empty
						return
					}
					if out, err := task(item); err != nil {
						errout <- err
						cancel()
					} else {
						if out != nil {
							output <- *out
						}
						wg.Done()
					}
				case <-ctx.Done():
					// Context cancelled, exit early
					return
				}
			}
		}()
	}

	wg.Wait()
	cancel()
	close(output)
	close(errout)

	itemsout := []T{}
	for out := range output {
		itemsout = append(itemsout, out)
	}

	err, haserr := <-errout
	if haserr {
		return itemsout, err
	}
	return itemsout, nil
}
