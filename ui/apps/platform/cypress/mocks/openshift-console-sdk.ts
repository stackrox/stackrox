/**
 * Mock implementation of @openshift-console/dynamic-plugin-sdk for Cypress component tests.
 * The SDK expects to run inside the OpenShift Console environment and will fail in test runners.
 */

export function useActiveNamespace(): [string | undefined, () => void] {
    return [undefined, () => {}];
}
