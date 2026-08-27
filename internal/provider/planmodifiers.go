package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dependentBoolPlanModifier plans an Optional+Computed bool whose API value
// only exists while a sibling attribute is set: `assertion_body_case_sensitive`
// is non-null exactly when `assertion_body_op` is. Left to
// UseStateForUnknown alone, removing the body assertion would plan the old
// `true` and the API would answer null, an "inconsistent result after apply".
type dependentBoolPlanModifier struct {
	sibling path.Path
}

func (m dependentBoolPlanModifier) Description(_ context.Context) string {
	return "Null when the sibling attribute is unset; otherwise keeps the prior state until the API answers."
}

func (m dependentBoolPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m dependentBoolPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	var sibling types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, m.sibling, &sibling)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if sibling.IsNull() {
		resp.PlanValue = types.BoolNull()
		return
	}
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		return
	}
	resp.PlanValue = types.BoolUnknown()
}
