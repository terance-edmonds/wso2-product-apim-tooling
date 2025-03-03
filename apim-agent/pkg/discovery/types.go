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

package discovery

// EventType represents the type of event.
type EventType string

const (
	// EventTypeCreate signifies a create event.
	EventTypeCreate EventType = "CREATE"
	// EventTypeUpdate signifies an update event.
	EventTypeUpdate EventType = "UPDATE"
	// EventTypeDelete signifies a delete event.
	EventTypeDelete EventType = "DELETE"
	applicationJSON           = "application/json"
	retryInterval             = 5
)

// APICRLabelsUpdate hold the label update required for a specific API CR
type APICRLabelsUpdate struct {
	Namespace string
	Name      string
	Labels    map[string]string
}

// APICPEvent represents data for the control plane API.
type APICPEvent struct {
	Event       EventType `json:"event"`
	API         API       `json:"payload"`
	CRName      string    `json:"-"`
	CRNamespace string    `json:"-"`
}

// API holds the data that needs to be sent to agent
type API struct {
	APIUUID              string            `json:"apiUUID"`
	APIName              string            `json:"apiName"`
	APIVersion           string            `json:"apiVersion"`
	IsDefaultVersion     bool              `json:"isDefaultVersion"`
	Definition           string            `json:"definition"`
	APIType              string            `json:"apiType"`
	APISubType           string            `json:"apiSubType"`
	BasePath             string            `json:"basePath"`
	Organization         string            `json:"organization"`
	SystemAPI            bool              `json:"systemAPI"`
	APIProperties        map[string]string `json:"apiProperties,omitempty"`
	Environment          string            `json:"environment,omitempty"`
	RevisionID           string            `json:"revisionID"`
	SandEndpoint         string            `json:"sandEndpoint"`
	SandEndpointSecurity EndpointSecurity  `json:"sandEndpointSecurity"`
	ProdEndpoint         string            `json:"prodEndpoint"`
	ProdEndpointSecurity EndpointSecurity  `json:"prodEndpointSecurity"`
	EndpointProtocol     string            `json:"endpointProtocol"`
	CORSPolicy           *CORSPolicy       `json:"cORSPolicy,omitempty"`
	Vhost                string            `json:"vhost"`
	SandVhost            string            `json:"sandVhost"`
	SecurityScheme       []string          `json:"securityScheme"`
	AuthHeader           string            `json:"authHeader"`
	APIKeyHeader         string            `json:"apiKeyHeader"`
	Operations           []Operation       `json:"operations"`
	AIConfiguration      AIConfiguration   `json:"aiConfiguration"`
	APIHash              string            `json:"-"`
	SandAIRL             *AIRL             `json:"sandAIRL"`
	ProdAIRL             *AIRL             `json:"prodAIRL"`
}

// AIRL holds AI ratelimit related data
type AIRL struct {
	PromptTokenCount     *uint32 `json:"promptTokenCount"`
	CompletionTokenCount *uint32 `json:"CompletionTokenCount"`
	TotalTokenCount      *uint32 `json:"totalTokenCount"`
	TimeUnit             string  `json:"timeUnit"`
	RequestCount         *uint32 `json:"requestCount"`
}

// EndpointSecurity holds the endpoint security information
type EndpointSecurity struct {
	Enabled       bool   `json:"enabled"`
	SecurityType  string `json:"securityType"`
	APIKeyName    string `json:"apiKeyName"`
	APIKeyValue   string `json:"apiKeyValue"`
	APIKeyIn      string `json:"apiKeyIn"`
	BasicUsername string `json:"basicUsername"`
	BasicPassword string `json:"basicPassword"`
}

// AIConfiguration holds the AI configuration
type AIConfiguration struct {
	LLMProviderID         string `json:"llmProviderID"`
	LLMProviderName       string `json:"llmProviderName"`
	LLMProviderAPIVersion string `json:"llmProviderAPIVersion"`
}

// Headers contains the request and response header modifier information
type Headers struct {
	RequestHeaders  HeaderModifier `json:"requestHeaders"`
	ResponseHeaders HeaderModifier `json:"responseHeaders"`
}

// HeaderModifier contains header modifier values
type HeaderModifier struct {
	AddHeaders    []Header `json:"addHeaders"`
	RemoveHeaders []string `json:"removeHeaders"`
}

// Header contains the header information
type Header struct {
	Name  string `json:"headerName"`
	Value string `json:"headerValue,omitempty"`
}

// Operation holds the path, verb, throttling and interceptor policy
type Operation struct {
	Path    string   `json:"path"`
	Verb    string   `json:"verb"`
	Scopes  []string `json:"scopes"`
	Headers Headers  `json:"headers"`
}

// CORSPolicy hold cors configs
type CORSPolicy struct {
	AccessControlAllowCredentials bool     `json:"accessControlAllowCredentials,omitempty"`
	AccessControlAllowHeaders     []string `json:"accessControlAllowHeaders,omitempty"`
	AccessControlAllowOrigins     []string `json:"accessControlAllowOrigins,omitempty"`
	AccessControlExposeHeaders    []string `json:"accessControlExposeHeaders,omitempty"`
	AccessControlMaxAge           *int     `json:"accessControlMaxAge,omitempty"`
	AccessControlAllowMethods     []string `json:"accessControlAllowMethods,omitempty"`
}
