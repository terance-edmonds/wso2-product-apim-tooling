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

package main

import (
	"os"

	"github.com/cucumber/godog"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/tests/pkg/utils"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/tests/steps"
)

func main() {
	ctx := utils.NewSharedContext()
	opts := godog.Options{
		Format: "progress",
		Paths:  []string{"features"},
	}

	status := godog.TestSuite{
		Name:                 "api_tests",
		TestSuiteInitializer: func(suiteContext *godog.TestSuiteContext) {},
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			steps.BaseSteps(s, ctx)
			steps.APIDeploymentSteps(s, ctx)
		},
		Options: &opts,
	}.Run()

	if status != 0 {
		os.Exit(status)
	}
}
