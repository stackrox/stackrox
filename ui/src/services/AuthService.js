import store from 'store';
import axios from 'axios';

let authProviders = [];

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
            .get('/v1/authProviders')
            .then(response => {
                const providers = response.data.authProviders;
                authProviders = providers;
                return { response: response.data };
            })
            .catch(error => console.error(error))
};

export default AuthService;
