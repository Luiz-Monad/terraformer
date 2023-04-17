package tfplugin

// this is the GRPC provider client that's missing from terraform-plugin-go@v0.15.0/tfprotov5

import (
	"context"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/configschema"
	fromproto "github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/fromproto"
	proto "github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/tfplugin5"
	toproto "github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/toproto"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"google.golang.org/grpc"
)

// function skeleton copied from terraform@v.1.4.5/internal/plugin/grpc_provider.go

// GRPCProviderPlugin implements plugin.GRPCPlugin for the go-plugin package.
type GRPCProviderPlugin struct {
	plugin.Plugin
	GRPCProvider func() proto.ProviderServer
}

func (p *GRPCProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &client{
		upstream: proto.NewProviderClient(c),
		ctx:      ctx,
	}, nil
}

func (p *GRPCProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

type client struct {
	tfprotov5.ProviderServer
	upstream proto.ProviderClient
	ctx      context.Context
	ClientContext
}

type ClientContext interface {
	Context() context.Context
}

func (c *client) Context() context.Context {
	return c.ctx
}

// the following implementation is the reverse of terraform-plugin-go@v0.15.0/tfprotov5/tf5server/server.go

func (c *client) GetProviderSchema(ctx context.Context, req *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error) {
	r, err := toproto.GetProviderSchema_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.GetSchema(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.GetProviderSchemaResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) PrepareProviderConfig(ctx context.Context, req *tfprotov5.PrepareProviderConfigRequest) (*tfprotov5.PrepareProviderConfigResponse, error) {
	r, err := toproto.PrepareProviderConfig_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.PrepareProviderConfig(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.PrepareProviderConfigResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) ConfigureProvider(ctx context.Context, req *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error) {
	r, err := toproto.Configure_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.Configure(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.ConfigureProviderResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) StopProvider(ctx context.Context, req *tfprotov5.StopProviderRequest) (*tfprotov5.StopProviderResponse, error) {
	r, err := toproto.Stop_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.Stop(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.StopProviderResponse(resp)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *client) ValidateDataSourceConfig(ctx context.Context, req *tfprotov5.ValidateDataSourceConfigRequest) (*tfprotov5.ValidateDataSourceConfigResponse, error) {
	r, err := toproto.ValidateDataSourceConfig_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.ValidateDataSourceConfig(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.ValidateDataSourceConfigResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) ReadDataSource(ctx context.Context, req *tfprotov5.ReadDataSourceRequest) (*tfprotov5.ReadDataSourceResponse, error) {
	r, err := toproto.ReadDataSource_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.ReadDataSource(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.ReadDataSourceResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) ValidateResourceTypeConfig(ctx context.Context, req *tfprotov5.ValidateResourceTypeConfigRequest) (*tfprotov5.ValidateResourceTypeConfigResponse, error) {
	r, err := toproto.ValidateResourceTypeConfig_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.ValidateResourceTypeConfig(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.ValidateResourceTypeConfigResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) UpgradeResourceState(ctx context.Context, req *tfprotov5.UpgradeResourceStateRequest) (*tfprotov5.UpgradeResourceStateResponse, error) {
	r, err := toproto.UpgradeResourceState_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.UpgradeResourceState(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.UpgradeResourceStateResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) ReadResource(ctx context.Context, req *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	r, err := toproto.ReadResource_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.ReadResource(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.ReadResourceResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) PlanResourceChange(ctx context.Context, req *tfprotov5.PlanResourceChangeRequest) (*tfprotov5.PlanResourceChangeResponse, error) {
	r, err := toproto.PlanResourceChange_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.PlanResourceChange(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.PlanResourceChangeResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) ApplyResourceChange(ctx context.Context, req *tfprotov5.ApplyResourceChangeRequest) (*tfprotov5.ApplyResourceChangeResponse, error) {
	r, err := toproto.ApplyResourceChange_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.ApplyResourceChange(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.ApplyResourceChangeResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}

func (c *client) ImportResourceState(ctx context.Context, req *tfprotov5.ImportResourceStateRequest) (*tfprotov5.ImportResourceStateResponse, error) {
	r, err := toproto.ImportResourceState_Request(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.upstream.ImportResourceState(ctx, r)
	if err != nil {
		return nil, err
	}
	ret, err := fromproto.ImportResourceStateResponse(resp)
	if err != nil {
		return nil, err
	}
	if w := configschema.WrapDiagnostics(ret.Diagnostics); w.HasError() {
		return ret, w.ToError()
	}
	return ret, nil
}
