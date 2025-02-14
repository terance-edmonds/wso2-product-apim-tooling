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

package transformer

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/config/types"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/loggers"
)

// GetUniqueIDForAPI will generate a unique ID for newly created APIs
func GetUniqueIDForAPI(name, version, organization string) string {
	concatenatedString := strings.Join([]string{organization, name, version}, "-")
	hash := sha1.New()
	hash.Write([]byte(concatenatedString))
	hashedValue := hash.Sum(nil)
	return hex.EncodeToString(hashedValue)
}

// GenerateOperationsMatrix generates a 2d array for the given number of operations
func GenerateOperationsMatrix(totalOperations int, maxColumns int) [][]types.Operation {
	rows := (totalOperations + maxColumns - 1) / maxColumns // Calculate the number of rows (ceil division)
	// Initialize operationsArray
	operationsArray := make([][]types.Operation, rows)
	remainingOperations := totalOperations
	for i := range operationsArray {
		columnsInRow := min(maxColumns, remainingOperations)
		operationsArray[i] = make([]types.Operation, columnsInRow)
		remainingOperations -= columnsInRow
	}
	return operationsArray
}

// GeneratePluginRefName generates a reference name for a plugin based on the operation, target reference, and plugin name.
func GeneratePluginRefName(operation *types.Operation, targetRef string, pluginName string) string {
	concatenatedString := pluginName
	if operation != nil {
		operationTargetHash := fmt.Sprintf("%x", sha1.Sum([]byte(operation.Target+operation.Verb)))
		concatenatedString = concatenatedString + "-" + operationTargetHash
		return "resource-" + concatenatedString + "-" + targetRef
	}
	serviceTargetHash := fmt.Sprintf("%x", sha1.Sum([]byte(pluginName+targetRef)))
	concatenatedString = concatenatedString + "-" + serviceTargetHash
	return "api-" + concatenatedString + "-" + targetRef
}

// GenerateConsumerName generates a reference name for a consumer
func GenerateConsumerName(applicationUUID string, consumerName string) string {
	consumerHash := fmt.Sprintf("%x", sha1.Sum([]byte(applicationUUID+consumerName)))
	return "consumer-" + consumerHash
}

// GenerateSecretName generates a reference name for a k8s secret
func GenerateSecretName(applicationUUID string, environment string, secretType string) string {
	return "secret-" + applicationUUID + "-" + strings.ToLower(environment) + "-" + secretType
}

// GenerateJSON converts go struct to json
func GenerateJSON(data KongPluginConfig) []byte {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		loggers.LoggerUtils.Errorf("Failed to generate json. Error: %v", err)
	}
	return jsonBytes
}

// generateSHA1Hash returns the SHA1 hash for the given string
func generateSHA1Hash(input string) string {
	h := sha1.New() /* #nosec */
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}
