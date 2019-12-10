/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package slices

// Index returns the element index of the string in the array
func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

// Contains returns true if the target string t is in the slice, and false otherwise.
func Contains(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

// Filter returns a list of strings from the input which satisfy the function.
func Filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

// RemoveDuplicates removes duplicates. Result is unordered
func RemoveDuplicates(vs []string) []string {
	dup := map[string]bool{}
	for e := range vs {
		dup[vs[e]] = true
	}

	var noDups []string
	for key := range dup {
		noDups = append(noDups, key)
	}
	return noDups
}
