export const systemConfigUrl = '/main/systemconfig';

const selectors = {
    navLinks: {
        config: '[data-testid="configure"]',
        subnavMenu: '[data-testid="configure-subnav"]',
        systemConfig: '[data-testid="system-config"]',
        topNav: '[data-testid="top-nav-btns"] button',
        logout: '[data-testid="Logout"]'
    },
    pageHeader: {
        editButton: '[data-testid="edit-btn"]',
        cancelButton: '[data-testid="cancel-btn"]',
        saveButton: '[data-testid="save-btn"]'
    },
    header: {
        widget: '[data-testid="header-config"]',
        state: '[data-testid="header-state"]',
        config: {
            toggle: '[data-testid="header-config"] .form-switch',
            textInput: '[data-testid="header-config"] textarea',
            colorPickerBtn: '[data-testid="header-config"] [data-testid="color-picker"]',
            colorInput: '[data-testid="header-config"] .chrome-picker input',
            size: {
                input: '[data-testid="header-config"] .react-select__input input',
                options: '[data-testid="header-config"] .react-select__option'
            }
        },
        banner: '[data-testid="header-banner"]'
    },
    footer: {
        widget: '[data-testid="footer-config"]',
        state: '[data-testid="footer-state"]',
        config: {
            toggle: '[data-testid="footer-config"] .form-switch',
            textInput: '[data-testid="footer-config"] textarea',
            colorPickerBtn: '[data-testid="footer-config"] [data-testid="color-picker"]',
            colorInput: '[data-testid="footer-config"] .chrome-picker input',
            size: {
                input: '[data-testid="footer-config"] .react-select__input input',
                options: '[data-testid="footer-config"] .react-select__option'
            }
        },
        banner: '[data-testid="footer-banner"]'
    },
    loginNotice: {
        widget: '[data-testid="login-notice-config"]',
        state: '[data-testid="login-notice-state"]',
        config: {
            toggle: '[data-testid="login-notice-config"] .form-switch',
            textInput: '[data-testid="login-notice-config"] textarea'
        },
        banner: '[data-testid="login-notice"]'
    }
};

export const text = {
    banner: 'Hello this is a sample banner text',
    color: '#000000',
    backgroundColor: '#ffff00'
};

export default selectors;
