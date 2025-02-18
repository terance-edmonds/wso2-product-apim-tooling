/*
 *  Copyright (c) 2024, WSO2 LLC. (http://www.wso2.org) All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */

package utils

import (
	"slices"
	"strings"
)

// FilterItems filter items
func FilterItems(items []string, filterItems []string) []string {
	result := []string{}
	for _, item := range items {
		if !slices.Contains(filterItems, item) {
			result = append(result, item)
		}
	}
	return result
}

// AddItems adds items to given string separated ","
func AddItems(items []string, addItems []string) []string {
	for _, item := range items {
		if !slices.Contains(addItems, item) {
			items = append(items, item)
		}
	}
	return items
}

// PrepareAnnotations adds/removes listed annotations from given list of annotations
func PrepareAnnotations(annotations string, items []string, remove bool) string {
	result := strings.Split(annotations, ",")
	if remove {
		result = FilterItems(result, items)
	} else {
		result = AddItems(result, items)
	}
	return strings.Join(result, ",")
}
