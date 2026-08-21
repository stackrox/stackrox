import * as yup from 'yup';

import type { DetailsType } from '../reports.types';

export const validationSchemaDetails: yup.ObjectSchema<DetailsType> = yup.object().shape({
    name: yup.string().trim().required('Report name is required'),
    description: yup.string().ensure(),
});
