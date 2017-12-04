import React from 'react';
import {
    BrowserRouter as Router,
    Route
} from 'react-router-dom';

import MainPage from 'Containers/MainPage';

const AppPage = () => (
    <Router>
        <Route path="/" component={MainPage} />
    </Router>
)

export default AppPage;
