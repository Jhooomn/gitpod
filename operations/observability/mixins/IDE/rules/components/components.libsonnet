/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the MIT License. See License-MIT.txt in the project root for license information.
 */

(import './openvsx-proxy/alerts.libsonnet')

{
  prometheusRules+:: {
    groups+: [],
    // IDE team doesn have any recording rules yet
  },
}
