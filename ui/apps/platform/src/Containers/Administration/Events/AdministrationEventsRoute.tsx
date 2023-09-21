import React, { ReactElement } from 'react';
import { useParams } from 'react-router-dom';

import AdministrationEventPage from './AdministrationEventPage';
import AdministrationEventsPage from './AdministrationEventsPage';

function AdministrationEventsRoute(): ReactElement {
    const { id } = useParams();

    if (id) {
        return <AdministrationEventPage id={id} />;
    }

    return <AdministrationEventsPage />;
}

export default AdministrationEventsRoute;
