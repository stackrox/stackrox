import withAuth from '../../../helpers/basicAuth';
import { visitListeningEndpoints } from './ListeningEndpoints.helpers';

describe('Listening endpoints page table', () => {
    withAuth();

    it('should render the listening endpoints audit page', () => {
        visitListeningEndpoints();
    });
});
