package toproto

import (
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/tfplugin5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

func GetProviderSchema_ServerCapabilities(in *tfprotov5.ServerCapabilities) *tfplugin5.GetProviderSchema_ServerCapabilities {
	if in == nil {
		return nil
	}

	return &tfplugin5.GetProviderSchema_ServerCapabilities{
		PlanDestroy: in.PlanDestroy,
	}
}
