import { gql } from '@apollo/client';

const GET_PROCESS_TAGS_COUNT = gql`
    query processTagsCount($key: ProcessNoteKey!) {
        processTagsCount(key: $key)
    }
`;

export default GET_PROCESS_TAGS_COUNT;
