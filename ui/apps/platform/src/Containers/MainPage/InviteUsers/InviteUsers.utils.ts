import { AuthProvider } from 'services/AuthService';

export type BucketsForNewAndExistingEmails = {
    newEmails: string[];
    existingEmails: string[];
};

export function splitEmailsIntoNewAndExisting(
    providerWithRules: AuthProvider,
    emailArr: string[]
): BucketsForNewAndExistingEmails {
    return emailArr.reduce<BucketsForNewAndExistingEmails>(
        (acc, email) => {
            if (
                Array.isArray(providerWithRules.groups) &&
                providerWithRules.groups.some(
                    (group) => group.props.key === 'email' && group.props.value === email
                )
            ) {
                return {
                    newEmails: acc.newEmails,
                    existingEmails: [...acc.existingEmails, email],
                };
            }
            return { newEmails: [...acc.newEmails, email], existingEmails: acc.existingEmails };
        },
        { newEmails: [], existingEmails: [] }
    );
}
