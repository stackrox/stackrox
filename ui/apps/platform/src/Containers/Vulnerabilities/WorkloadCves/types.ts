export type Severity = 'Critical' | 'Important' | 'Moderate' | 'Low';
export type FixableStatus = 'Fixable' | 'Not fixable';

export type DefaultFilters = {
    Severity: Severity[];
    Fixable: FixableStatus[];
};

export type VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: DefaultFilters;
    };
};
