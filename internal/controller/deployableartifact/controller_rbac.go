/*
 * Copyright (c) 2025, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package deployableartifact

// RBAC annotations for the deployableartifact controller are defined in this file.

// +kubebuilder:rbac:groups=core.choreo.dev,resources=deployableartifacts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.choreo.dev,resources=deployableartifacts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.choreo.dev,resources=deployableartifacts/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
