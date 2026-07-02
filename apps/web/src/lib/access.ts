import type { PermissionRequirement } from './permissions'

export const adminShellAccess: PermissionRequirement = {
  authorities: [
    'admin',
    'super_admin',
    'system:admin',
    'knowledge:admin',
    'knowledge:write',
    'admin:model-profile:write',
    'admin:parser-config:write',
    'report:read',
    'report:write',
    'reports:write',
  ],
}
