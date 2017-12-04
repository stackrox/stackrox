import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import AppPage from './Containers/AppPage';
import registerServiceWorker from './registerServiceWorker';

ReactDOM.render(<AppPage />, document.getElementById('root'));

registerServiceWorker();
