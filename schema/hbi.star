load("rbac.star", "v1_based_permission")
load("kessel.star", "relation", "assignable", "cardinality", "subref", "union", "field", "uuid", "text")

# Everything in this file is reporter = hbi

# Creates resource type 'host' for report/check purposes
host = {
    # Assign-only relation results in a field of the given type (uuid) that can be reported and maps to an rbac workspace
    "workspace": relation(assignable("rbac", "workspace", cardinality.ExactlyOne, uuid())),

    # Data fields - reported with their names and given types
    "satellite_id": field(type=union(uuid(), text(regex="^\\d{10}$"))), #This field uses a type union - can be uuid or 10-digit character string
    "subscription_manager_id": field(type=uuid()),
    "insights_id": field(type=uuid()),
    "ansible_host": field(type=text(maxLength=255)),

    # Readonly relations - not reportable, names are used for check and lookup
    # The v1_based_permission function (imported from rbac) maps a permission from V1 RBAC down to the workspace and returns the permission name, allowing it to be nested in a subref call as below
    "view": relation(subref("workspace", v1_based_permission("inventory", "host", "read", "inventory_host_view"))), #Maps inventory:host:read to inventory_host_view at the workspace level and returns it
                                                                                                                    #'view' then lights up when workspace.inventory_host_view does

    "update": relation(subref("workspace", v1_based_permission("inventory", "host", "write", "inventory_host_update")))
}