package fromproto

import (
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/tfplugin5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

func RawState(in *tfplugin5.RawState) *tfprotov5.RawState {
	return &tfprotov5.RawState{
		JSON:    in.Json,
		Flatmap: in.Flatmap,
	}
}
