import * as yup from 'yup';

export const maxCustomSubjectLength = 256;
export const maxCustomBodyLength = 1500;

export const customSubjectValidation = yup
    .string()
    .default('')
    .max(
        maxCustomSubjectLength,
        `Limit your input to ${maxCustomSubjectLength} characters or fewer.`
    );

export const customBodyValidation = yup
    .string()
    .default('')
    .max(maxCustomBodyLength, `Limit your input to ${maxCustomBodyLength} characters or fewer.`);

export const emailTemplateValidationSchema = yup.object({
    customSubject: customSubjectValidation,
    customBody: customBodyValidation,
});

export type EmailTemplateFormData = yup.InferType<typeof emailTemplateValidationSchema>;

export function isDefaultEmailTemplate({
    customBody,
    customSubject,
}: EmailTemplateFormData): boolean {
    return customBody.length === 0 && customSubject.length === 0;
}
