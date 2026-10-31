import * as yup from 'yup';

const maxCustomSubjectLength = 256;
const maxCustomBodyLength = 1500;

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
