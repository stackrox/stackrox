import { useQuery } from '@tanstack/react-query';

import { fetchPermissionSets, fetchRolesAsArray } from 'services/RolesService';
import { fetchAuthProviders } from 'services/AuthService/AuthService';
import { fetchAccessScopes } from 'services/AccessScopesService';

import { Card, CardHeader, CardTitle, CardContent } from 'design-system/ui/card';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';

function StatCard({
    title,
    count,
    isLoading,
}: {
    title: string;
    count?: number;
    isLoading: boolean;
}) {
    return (
        <Card>
            <CardContent className="pt-4">
                {isLoading ? (
                    <Skeleton className="h-10 w-full" />
                ) : (
                    <div className="text-center">
                        <div className="font-mono text-3xl font-700 text-text-primary">
                            {count ?? 0}
                        </div>
                        <div className="mt-1 text-xs text-text-muted">{title}</div>
                    </div>
                )}
            </CardContent>
        </Card>
    );
}

export default function AccessControlPageCC() {
    const { data: roles, isLoading: rolesLoading } = useQuery({
        queryKey: ['access-control', 'roles'],
        queryFn: () => fetchRolesAsArray(),
    });

    const { data: authProviders, isLoading: authLoading } = useQuery({
        queryKey: ['access-control', 'auth-providers'],
        queryFn: async () => {
            const result = await fetchAuthProviders();
            return result.response;
        },
    });

    const { data: permSets, isLoading: permLoading } = useQuery({
        queryKey: ['access-control', 'permission-sets'],
        queryFn: () => fetchPermissionSets(),
    });

    const { data: scopes, isLoading: scopesLoading } = useQuery({
        queryKey: ['access-control', 'access-scopes'],
        queryFn: () => fetchAccessScopes(),
    });

    return (
        <>
            <div className="p-5">
                <div className="mb-4">
                    <h2 className="text-sm font-600 text-text-primary">Access Control Overview</h2>
                    <p className="mt-1 text-xs text-text-muted">
                        Manage authentication providers, roles, permission sets, and access scopes.
                    </p>
                </div>

                <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                    <StatCard
                        title="Auth Providers"
                        count={authProviders?.length}
                        isLoading={authLoading}
                    />
                    <StatCard title="Roles" count={roles?.length} isLoading={rolesLoading} />
                    <StatCard
                        title="Permission Sets"
                        count={permSets?.length}
                        isLoading={permLoading}
                    />
                    <StatCard
                        title="Access Scopes"
                        count={scopes?.length}
                        isLoading={scopesLoading}
                    />
                </div>

                {/* Roles table */}
                {!rolesLoading && roles && (
                    <div className="mt-6">
                        <Card>
                            <CardHeader>
                                <CardTitle>Roles</CardTitle>
                                <Badge variant="info">{roles.length}</Badge>
                            </CardHeader>
                            <CardContent>
                                <table className="w-full border-collapse">
                                    <thead>
                                        <tr>
                                            <th className="pb-1.5 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                                Name
                                            </th>
                                            <th className="pb-1.5 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                                Permission Set
                                            </th>
                                            <th className="pb-1.5 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                                Access Scope
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {roles.map((role) => (
                                            <tr
                                                key={role.name}
                                                className="border-b border-border-subtle last:border-b-0 hover:bg-bg-hover"
                                            >
                                                <td className="py-2 text-xs text-text-primary">
                                                    {role.name}
                                                </td>
                                                <td className="py-2 font-mono text-2xs text-text-secondary">
                                                    {role.permissionSetId || '—'}
                                                </td>
                                                <td className="py-2 font-mono text-2xs text-text-secondary">
                                                    {role.accessScopeId || '—'}
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </CardContent>
                        </Card>
                    </div>
                )}
            </div>
        </>
    );
}
