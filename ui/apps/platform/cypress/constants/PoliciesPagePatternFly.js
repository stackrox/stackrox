export const url = '/main/policies-pf';

export const selectors = {
    table: {
        trFirst: 'tbody tr:nth-child(1)',
        linkToPolicy: 'td[data-label="Policy"] a',
        buttonForActionToggle: 'td.pf-c-table__action button.pf-c-dropdown__toggle',
        buttonForActionItem: 'td.pf-c-table__action ul li[role="menuitem"] button',
    },
    page: {
        buttonForActionToggle: 'button.pf-c-dropdown__toggle:contains("Actions")',
        buttonForActionItem:
            'button.pf-c-dropdown__toggle:contains("Actions") + ul li[role="menuitem"] button',
    },
    toast: {
        title: 'ul.pf-c-alert-group .pf-c-alert__title',
        description: 'ul.pf-c-alert-group .pf-c-alert__description',
    },
    wizard: {},
};
