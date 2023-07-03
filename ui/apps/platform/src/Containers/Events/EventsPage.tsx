import React, { ReactElement } from 'react';
import { Route } from 'react-router-dom';
import { eventsPath } from '../../routePaths';
import EventsListPage from './EventsTablePage';

function EventsPage(): ReactElement {
    return <Route path={eventsPath} component={EventsListPage} />;
}

export default EventsPage;
