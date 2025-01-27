package toproto

import (
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/tfplugin5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

func ValidateDataSourceConfig_Request(in *tfprotov5.ValidateDataSourceConfigRequest) (*tfplugin5.ValidateDataSourceConfig_Request, error) {
	resp := &tfplugin5.ValidateDataSourceConfig_Request{
		TypeName: in.TypeName,
	}
	if in.Config != nil {
		resp.Config = DynamicValue(in.Config)
	}
	return resp, nil
}

func ValidateDataSourceConfig_Response(in *tfprotov5.ValidateDataSourceConfigResponse) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	diags, err := Diagnostics(in.Diagnostics)
	if err != nil {
		return nil, err
	}
	return &tfplugin5.ValidateDataSourceConfig_Response{
		Diagnostics: diags,
	}, nil
}

func ReadDataSource_Request(in *tfprotov5.ReadDataSourceRequest) (*tfplugin5.ReadDataSource_Request, error) {
	resp := &tfplugin5.ReadDataSource_Request{
		TypeName: in.TypeName,
	}
	if in.Config != nil {
		resp.Config = DynamicValue(in.Config)
	}
	if in.ProviderMeta != nil {
		resp.ProviderMeta = DynamicValue(in.ProviderMeta)
	}
	return resp, nil
}

func ReadDataSource_Response(in *tfprotov5.ReadDataSourceResponse) (*tfplugin5.ReadDataSource_Response, error) {
	diags, err := Diagnostics(in.Diagnostics)
	if err != nil {
		return nil, err
	}
	resp := &tfplugin5.ReadDataSource_Response{
		Diagnostics: diags,
	}
	if in.State != nil {
		resp.State = DynamicValue(in.State)
	}
	return resp, nil
}

// we have to say this next thing to get golint to stop yelling at us about the
// underscores in the function names. We want the function names to match
// actually-generated code, so it feels like fair play. It's just a shame we
// lose golint for the entire file.
//
// This file is not actually generated. You can edit it. Ignore this next line.
// Code generated by hand ignore this next bit DO NOT EDIT.
