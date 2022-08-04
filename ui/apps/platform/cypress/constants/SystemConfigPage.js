import navigationSelectors from '../selectors/navigation';

export const systemConfigUrl = '/main/systemconfig';

const selectors = {
    navLinks: {
        configure: `${navigationSelectors.navExpandable}:contains("Platform Configuration")`,
        subnavMenu: '[data-testid="configure-subnav"]',
        systemConfig: `${navigationSelectors.navLinks}:contains("System Configuration")`,
        topNav: '[aria-label="User menu"]',
        logout: '.pf-c-page__header-tools-item button:contains("Log out")',
    },
    pageHeader: {
        editButton: 'button:contains("Edit")',
        cancelButton: 'button:contains("Cancel")',
        saveButton: 'button:contains("Save")',
    },
    header: {
        widget: '[data-testid="header-config"]',
        state: '[data-testid="header-state"]',
        config: {
            toggle: '[data-testid="header-config"] .pf-c-switch input',
            textInput: '[data-testid="header-config"] textarea',
            colorPickerBtn: '[data-testid="header-config"] [data-testid="color-picker"]',
            colorInput: '[data-testid="header-config"] .chrome-picker input',
            size: {
                input: '[data-testid="header-config"] .pf-c-select button',
                options: '[data-testid="header-config"] .pf-c-select__menu li',
            },
        },
        banner: '[data-testid="header-banner"]',
    },
    footer: {
        widget: '[data-testid="footer-config"]',
        state: '[data-testid="footer-state"]',
        config: {
            toggle: '[data-testid="footer-config"] .pf-c-switch input',
            textInput: '[data-testid="footer-config"] textarea',
            colorPickerBtn: '[data-testid="footer-config"] [data-testid="color-picker"]',
            colorInput: '[data-testid="footer-config"] .chrome-picker input',
            size: {
                input: '[data-testid="footer-config"] .pf-c-select button',
                options: '[data-testid="footer-config"] .pf-c-select__menu li',
            },
        },
        banner: '[data-testid="footer-banner"]',
    },
    loginNotice: {
        widget: '[data-testid="login-notice-config"]',
        state: '[data-testid="login-notice-state"]',
        config: {
            toggle: '[data-testid="login-notice-config"] .pf-c-switch input',
            textInput: '[data-testid="login-notice-config"] textarea',
        },
        banner: '[data-testid="login-notice"]',
    },
    dataRetention: {
        widget: '[data-testid="data-retention-config"]',
        allRuntimeViolationsBox: '.pf-c-card:contains("All runtime violations")',
        deletedRuntimeViolationsBox:
            '.pf-c-card:contains("Runtime violations for deleted deployments")',
        resolvedDeployViolationsBox: '.pf-c-card:contains("Resolved deploy-phase violations")',
        imagesBox: '.pf-c-card:contains("Images no longer deployed")',
    },
};

export const text = {
    banner: 'Hello this is a sample banner text',
    color: '#000000',
    backgroundColor: '#ffff00',
};

export default selectors;
