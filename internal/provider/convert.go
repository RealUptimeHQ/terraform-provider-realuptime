package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/client"
)

// Small conversions between framework values and the API client's pointer
// fields. Kept in one place so "null in Terraform" <-> "null in the API" is
// decided once.

func strPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

func boolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func fromStrPtr(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

func fromInt64Ptr(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

func fromBoolPtr(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}

// nullable maps a planned string attribute onto a PATCH field: unknown or
// null means "clear" when the attribute was previously set (the caller
// decides whether to send by comparing plan and state), otherwise the value.
func nullable(v types.String) client.NullableString {
	if v.IsNull() || v.IsUnknown() {
		return client.Null()
	}
	return client.Str(v.ValueString())
}

func stringSetFrom(ctx context.Context, v types.Set) ([]string, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := v.ElementsAs(ctx, &out, false)
	return out, diags
}

func stringSetValue(items []string) types.Set {
	elems := make([]attr.Value, 0, len(items))
	for _, s := range items {
		elems = append(elems, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, elems)
}
