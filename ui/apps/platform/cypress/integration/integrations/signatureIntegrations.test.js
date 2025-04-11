import withAuth from '../../helpers/basicAuth';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';

import {
    clickCreateNewIntegrationInTable,
    clickIntegrationSourceLinkInForm,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    visitIntegrationsTable,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'signatureIntegrations';

describe('Signature Integrations', () => {
    withAuth();

    it('should create a new signature integration and then view and delete', () => {
        const integrationName = generateNameWithDate('Signature Integration');
        const integrationType = 'signature';

        const publicKeyValue = `-----BEGIN PUBLIC KEY-----
MIIBigKCAYEAnLceC91dTu1Lj6pMcLL3zcmps+NkczJPIaHDn8OtEnj+XzdmsMjO
zzmROtVH1HnsvDn5/tlxfqCMbWY1E6ezuj8wY9GY6eGHvEjU8JdZUw0Zoy2m3OV0
L3PDEuzATyT0fUjUNgjSXLNLLNl2LEF9yw/UP7QiHhj1mLojGUjaQ1REzBqkfsP2
7vR4AQbbf77/b5dwisoDYZXa+RnJ8IHWtXlnkBbf8eTo+8EArMGexpznSC4F5aL+
3aPl3Y2MFdmW2rDvjy4gNQQtBquJDIoyZEMTlDbMH4WV+44fZZfw0AP5MGPj1y+h
I1ea2UeFSkCWz+BDGHCj0kIUwLcDZaZfT4lu5qNe6XuEeTpPjnrEbqPf3NGg0DLQ
ZSpZ6ih3oWto2uTknM1Tf97Nr41J6nqec6Auott3oE9ww5KiJEiVi9q9L7cMupmS
xPP9jtUUiPdAw4uL71gLncP/YRYYyvjH3/aveFSlc83mS808FTRHiNfwBKHppuLW
HS1I6y+PPPrVAgMBAAE=
-----END PUBLIC KEY-----
`;

        const certificateOIDCIssuer = 'testing';
        const certificateIdentity = '.*';

        const chainPEM = `-----BEGIN CERTIFICATE-----
MIIFKTCCAxGgAwIBAgIUKw8Mzk3KbVr8avxpUA6g8KY1m2MwDQYJKoZIhvcNAQEL
BQAwJDEQMA4GA1UECgwHdGVzdGluZzEQMA4GA1UEAwwHdGVzdGluZzAeFw0yNDA0
MjEyMjM5NDRaFw0yNzAyMDkyMjM5NDRaMCQxEDAOBgNVBAoMB3Rlc3RpbmcxEDAO
BgNVBAMMB3Rlc3RpbmcwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCB
wg+zvNVik+XlGBIEUZiWAIe8hsxujlqxVrDPojjv0UiwVg6ho737JOyc11ruu4nd
TOjpf0PfLCJBm9Acsex9utyfUoLNviE63/Ivw+UgzLzvKZVB9hcLrVLQbUkpY6mJ
YTzvrJoxaacKx07agzRIOOM0sbRhte9xyKVvSIi7aj9FSnsvix7/3+eZoYp0grdt
umakcGcxklQh+tsXUBmA2w6QG/hkSwPh9ciOLiabiwndc5PW0w2QaFy3fzZ0iCoD
X4C+1srpVXLguJkQ7UaeTDOywKQ4e+sjM88iZrWHOFp2Vo2sTobz8bpzBYk3Ce/g
OoU7jBQIJF0BTTarQCgZWKxGppMbg4EmC6H5ml8QXbpDFOi3MQsJQnCM8xnImPL1
0Upgoko+dKq/fb0YJYbk2AeE+D7IsxVB8vgKLRk4YsSweO1VV3h8t/VTKHMLFSGr
RosUIOL0f18/tUqP00PeVjcL+Sp5tqacwRq891rjfZPgl3GywlTZhJ5lHCk531SV
uPelqBI5v+WqZwIyTLGkRx56n+gkBUmKjGfgEm6Y+pzXOffGIwMOlUFx4ur7k/iS
3TF6H1xc3EaUYSmtEbVvBo89AKyUYj2FEWx/Tq1SIZoJeZYowo+2t6i57JOf3pq6
y15zYpqXCHfe2/lw5Wk11RmlFcXXLALUWK4yaG8q7wIDAQABo1MwUTAdBgNVHQ4E
FgQUgulfifNv2AP154rAlWIQdN54diwwHwYDVR0jBBgwFoAUgulfifNv2AP154rA
lWIQdN54diwwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAgEAR+Yz
+wLAOzYkHJRqii90o1u4N8MUGsNhy53oFx8OZpUq1jflEO4U9jH1vceJodCkh//U
7CH9Pa4XZdkIif0IvFducW0Jw/cHmj/XMqOGcmQlhYxUsgyC1XSJjX4QsG1T17Ps
mdoustIktkRJR3OMXGIboQcj9cK7Wqvk8RTTRf4y4ekn1YZsJ7GGrSXoQb9Ln0KF
uk7ZxVuxZsQKxbMJOKBazSDmeAeCXGSBgdci8msw1qIRAQhiDLYiohrSRP8RIExY
9X+Fun9RKtJVFo7v0OMOoi7c2jewNTcCXYq9bL5KjR+s+EYigExhJRfCZAypUCHZ
Kh6nuOGoZboNgHcNvXjwBeJxqUqS0gx8fzgMz78z+az5Hf/B5SFtgf+Sl+NVgvaB
3pcOp3W6OjT1o7tiI6PjmwrcjgsIxkgQAzfebkXlDzxbwZV97YayxdyV3bDRl6r8
O9SWwxUFdCAHjJG2mEqmLI03W7YDV4Sx3XzfsJWw4BYm8RPTFfzeB8nK0aeLH8rJ
6tEq9pb2qBK9FwPZVpf+RSPXKcIwzdC+D0XXj3DbxB0RMnf6VWlsRrCCLL3W0qaf
yeoSmxEp+ol9zZcbDm7675FsmACMTikiKYYcIUJv0Ld4X+rHKH1ncd5o1rUQObFc
DxSQWhyf58r/eUY3CY2/SaoVBYtwbBICDw6PD7U=
-----END CERTIFICATE-----
`;

        const certPEM = `-----BEGIN CERTIFICATE-----
MIIEGDCCAgCgAwIBAgIUJk1lM6fPU8kjiZa5IvXhzK3V/+MwDQYJKoZIhvcNAQEL
BQAwJDEQMA4GA1UECgwHdGVzdGluZzEQMA4GA1UEAwwHdGVzdGluZzAeFw0yNDA0
MjEyMjQwNDBaFw0yNTA5MDMyMjQwNDBaMCQxEDAOBgNVBAoMB3Rlc3RpbmcxEDAO
BgNVBAMMB3Rlc3RpbmcwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDJ
Ach0Q/wXrkivOJ6O9c43EGltWgbd7EBATOQ1SP3J4IUGpCHfSWp7Mnnp1Kfo7fiS
LrSvBAQgf1+IIOlRR9qb4Dz1zkv5JIi2Bsx3CY5Z7LR49o1h3GwKdNzg/57wOrm6
8xkWv9Eh/2rV4XU4h2uLR0F3iZvJ+roT4JtMAJRIU/9cUr8zHwfBfiJEbFK+KxZW
QJMKRg7NYup4vbsDrJplLKUFXdwd5kq7RM059W7VdbWSbsPXONmKGOimLXKew7Vh
6ZAxxj2dZC88oocvGJmtn4Bg46/LTqU+81ot7fUDeRluoXUIGjUmUDsYWTj9i81k
k9/0pVKNrRsVd69yxHWPAgMBAAGjQjBAMB0GA1UdDgQWBBT8QfcgemDIYq5sD6tn
0bFYYq+vlTAfBgNVHSMEGDAWgBSC6V+J82/YA/XnisCVYhB03nh2LDANBgkqhkiG
9w0BAQsFAAOCAgEAWOr1xyZ+YKaZUAPSCmfA9BwIFACrNnkmm5HiY1lU7Yhs0Xgr
q9ed115I5ixOk5QR6YlHy3xnC4aNHyPUlxXefIWTELm1s3Ii0Dm7SrAXfM5iyHrG
YKBpyV320P4udnfBhEVL3kL3xxk23jQJzfAHJCMNLtms1V4XqXun7tMv5tMukCgk
RC9Y/grAK/1m13KKQNyMoRPqp+qBZmuMSwSliNNpZgb6BhljiyUJ4UZnZr6irRTe
Wu4nnqZtX1qqxrgKuF68f5jBKwOxRIZ0BJaCSaGlLGL4en8CNYd6TAE/OC4P4Zpz
18EDZAejZ4rS5tDEGtpDpHD2XeCeQACt/joJMTCwmePJEH73VF+ZFywbMNIAMsPR
mFESys6J7jqoSC9lQPjNoC2KbRnk+PwQ0U7NTJLJETGsWYhFHDwTi7g0Ogmwr09A
f4vOwI6+qmsckq2K3lob7VmdLhfzVy+u+q6Cg5eBHHyePoO5qtI/Zk9x4paUgMmE
74MTUknrnOm0GMGHMAyJKqsWZcfGWpLZf2TVFKu1MLcj3wx2Q7TFsqomcZW4Jlez
tZFwJgN+v0YobsrJlWjcS8vK6hWcMSyHoX+wVvMckaab0ycjTuYpybSuQG5G+002
I8+UmQtwa0MOOcoUeXXJXjGagodO6A22hzjwQyf5e87eeLA1FfwtGYNLjoA=
-----END CERTIFICATE-----
`;

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        // Check initial state.
        cy.get(selectors.buttons.save).should('be.disabled');

        // Check empty values are not accepted.
        getInputByLabel('Integration name').type(' ');
        cy.get('button:contains("Cosign public keys")').click({ force: true });
        cy.get('button:contains("Add new public key")').click({ force: true });
        getInputByLabel('Public key name').type('  ');
        getInputByLabel('Public key value').type('  ');
        getHelperElementByLabel('Integration name').contains('Integration name is required');
        getHelperElementByLabel('Public key name').contains('Name is required');

        cy.get('button:contains("Cosign certificates")').click({ force: true });
        cy.get('button:contains("Add new certificate verification")').click({ force: true });
        getInputByLabel('Certificate OIDC issuer').type('  ');
        getInputByLabel('Certificate identity').type('  ');
        getInputByLabel('Certificate chain (PEM encoded)').type('  ');
        getHelperElementByLabel('Certificate OIDC issuer').contains(
            'Certificate OIDC issuer is required'
        );
        getHelperElementByLabel('Certificate identity').contains(
            'Certificate identity is required'
        );
        cy.get(selectors.buttons.save).should('be.disabled');

        // Check conditional states.
        getInputByLabel('Enable transparency log validation').uncheck();
        getInputByLabel('Enable certificate transparency log validation').uncheck();

        getHelperElementByLabel('Rekor URL').should('be.disabled');
        getInputByLabel('Validate in offline mode').should('be.disabled');
        getInputByLabel('Rekor public key').should('be.disabled');
        getInputByLabel('Certificate transparency log public key').should('be.disabled');

        // Save integration.
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Public key name').clear().type('keyName');
        getInputByLabel('Public key value').clear().type(publicKeyValue);
        getInputByLabel('Certificate OIDC issuer').clear().type(certificateOIDCIssuer);
        getInputByLabel('Certificate identity').clear().type(certificateIdentity);
        getInputByLabel('Certificate chain (PEM encoded)').clear().type(chainPEM);
        getInputByLabel('Intermediate certificate (PEM encoded)').clear().type(certPEM);
        getInputByLabel('Enable transparency log validation').check();
        getInputByLabel('Enable certificate transparency log validation').check();

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        // View it.

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).click();

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationName}")`);

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Delete it.

        deleteIntegrationInTable(integrationSource, integrationType, integrationName);

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).should('not.exist');
    });
});
