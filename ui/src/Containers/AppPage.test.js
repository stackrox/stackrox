import React from 'react';
import { shallow } from 'enzyme';

import AppPage from 'Containers/AppPage';

it('renders the root app page without crashing', () => {
    shallow(<AppPage />);
});
