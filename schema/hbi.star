load("rbac.star", "v1_based_permission")
load("kessel.star", "relation", "assignable", "cardinality", "subref", "union", "field", "uuid", "text") #TODO: can we make these builtins or something?

host = {
    "workspace": relation(assignable("rbac", "workspace", cardinality.ExactlyOne, uuid())),

    "satellite_id": field(type=union(uuid(), text(regex="^\\d{10}$"))),
    "subscription_manager_id": field(type=uuid()),
    "insights_id": field(type=uuid()),
    "ansible_host": field(type=text(maxLength=255)),

    "view": relation(subref("workspace", v1_based_permission("inventory", "host", "read", "inventory_host_view"))),
    "update": relation(subref("workspace", v1_based_permission("inventory", "host", "write", "inventory_host_update")))
}