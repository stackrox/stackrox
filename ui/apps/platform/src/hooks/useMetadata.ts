import { useSelector } from 'react-redux';

import { selectors } from 'reducers';
import { Metadata } from 'types/metadataService.proto';

function useMetadata(): Metadata {
    const metadata = useSelector(selectors.metadataSelector);

    return metadata;
}

export default useMetadata;
