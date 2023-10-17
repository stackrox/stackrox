import { isAfter } from 'date-fns';
import * as yup from 'yup';

export type ScopeContext = 'GLOBAL' | { image: { name: string; tag: string } };

export const deferralValidationSchema = yup.object({
    cves: yup.array().of(yup.string()).min(1, 'At least one CVE must be selected'),
    comment: yup.string().required('A rationale is required'),
});

export type DeferralValues = {
    cves: string[];
    comment: string;
    scope: {
        imageScope: {
            registry: string;
            remote: string;
            tag: string;
        };
    };
    expiry?:
        | {
              type: 'TIME';
              days: number;
          }
        | {
              type: 'ALL_CVE_FIXABLE' | 'ANY_CVE_FIXABLE' | 'INDEFINITE';
          }
        | {
              type: 'CUSTOM_DATE';
              date: string;
          };
};

export function futureDateValidator(date: Date): string {
    return isAfter(new Date(), date) ? 'Date must be in the future' : '';
}
