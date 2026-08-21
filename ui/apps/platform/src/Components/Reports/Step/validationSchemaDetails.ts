import * as yup from 'yup';

import type { DetailsType } from 'Components/Reports/reports.types'; // TODO ./

export const validationSchemaDetails: yup.ObjectSchema<DetailsType> = yup.object().shape({
    name: yup.string().trim().required('Report name is required'),
    description: yup.string().ensure(),
});
