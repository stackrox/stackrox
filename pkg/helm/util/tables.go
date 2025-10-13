package util

import "helm.sh/helm/v3/pkg/chartutil"

// CoalesceTables is a variadic version of chartutil.CoalesceTables from the official Helm libraries.
// It combines an arbitrary number of tables, modifying the first argument (`dst`) and giving preference
// to arguments in left-to-right order.
// Hence, `CoalesceTables(dst, src1, src2, ..., srcN)` is equivalent to calling
//
//	CoalesceTables(...CoalesceTables(CoalesceTables(dst, src1), src2)..., srcN)
func CoalesceTables(dst map[string]interface{}, srcs ...map[string]interface{}) map[string]interface{} {
	res := dst
	for _, src := range srcs {
		res = chartutil.CoalesceTables(res, src)
	}
	return res
}
