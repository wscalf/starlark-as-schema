load("rbac.star", "v1_based_permission")

v1_based_permission("remediations", "remediations", "read", "remediations_remediation_view")
v1_based_permission("remediations", "remediations", "write", "remediations_remediation_update")