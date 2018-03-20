import store from 'store';
import axios from 'axios';

let authProviders = [];
const authProvidersUrl = '/v1/authProviders';

const AuthService = {
    login: token => {
        store.set('access_token', token);
    },
    logout: () => {
        store.remove('access_token');
    },
    isLoggedIn: () => AuthService.getAccessToken(),
    getAccessToken: () => store.get('access_token'),
    getAuthProviders: () => authProviders,
    updateAuthProviders: () =>
        axios
            .get(authProvidersUrl)
            .then(response => {
                const providers = response.data.authProviders;
                authProviders = providers;
                return { response: response.data };
            })
            .catch(error => console.error(error)),
    saveAuthProviders: data =>
        data.id !== undefined && data.id !== ''
            ? axios.put(`${authProvidersUrl}/${data.id}`, data)
            : axios.post(authProvidersUrl, data),
    deleteAuthProviders: data => axios.delete(`${authProvidersUrl}/${data.id}`)
};

export default AuthService;
