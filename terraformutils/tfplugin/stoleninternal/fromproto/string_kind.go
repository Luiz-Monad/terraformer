package fromproto

import (
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/tfplugin/stoleninternal/tfplugin5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

func StringKind(in tfplugin5.StringKind) tfprotov5.StringKind {
	return tfprotov5.StringKind(in)
}
