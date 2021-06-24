/* eslint-disable import/prefer-default-export */
import { AuthProvider } from 'services/AuthService';

export interface DisplayedAuthProvider extends AuthProvider {
    do_not_use_client_secret?: boolean;
}

function transformInitialValues(initialValues: DisplayedAuthProvider): DisplayedAuthProvider {
    // TODO-ivan: eventually logic for different auth provider type should live
    // with the form component that renders form for the corresponding auth provider
    // type, probably makes sense to refactor after moving away from redux-form
    if (initialValues.type === 'oidc') {
        const alteredConfig = { ...initialValues.config };

        // backend doesn't return the exact value for the client secret for the security reasons,
        // instead it'll return some obfuscated data, but not an empty one
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        alteredConfig.clientOnly = {
            clientSecretStored: !!alteredConfig.client_secret,
        };

        if (initialValues.name) {
            // if it's an existing auth provider, then we're using the secret if we have it
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            alteredConfig.do_not_use_client_secret = !alteredConfig.client_secret;
        }

        // clean-up obfuscated value if any as we don't need to show it
        alteredConfig.client_secret = '';

        return {
            ...initialValues,
            config: alteredConfig,
        };
    }
    if (initialValues.type === 'saml') {
        const alteredConfig = { ...initialValues.config };
        // unless static config values are present, assume dynamic configuration is selected
        alteredConfig.type = alteredConfig.idp_issuer ? 'static' : 'dynamic';
        return {
            ...initialValues,
            config: alteredConfig,
        };
    }
    return initialValues;
}

function populateDefaultValues(authProvider: AuthProvider): AuthProvider {
    const newInitialValues: DisplayedAuthProvider = { ...authProvider };
    newInitialValues.uiEndpoint = window.location.host;
    newInitialValues.enabled = true;
    if (authProvider.type === 'oidc') {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        newInitialValues.config = { mode: 'auto', do_not_use_client_secret: false };
    }
    return newInitialValues;
}

export function getInitialAuthProviderValues(authProvider: AuthProvider): DisplayedAuthProvider {
    const initialValues: DisplayedAuthProvider =
        !authProvider.name && !authProvider.id
            ? populateDefaultValues(authProvider)
            : { ...authProvider };

    const modifiedInitialValues = {
        ...transformInitialValues(initialValues),
    };

    return modifiedInitialValues;
}
