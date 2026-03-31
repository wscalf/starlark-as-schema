load("rbac.star", "v1_based_permission")

v1_based_permission("tasks", "tasks", "read", "tasks_task_view")
v1_based_permission("tasks", "tasks", "write", "tasks_task_update")
