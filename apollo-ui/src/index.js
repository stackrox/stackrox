import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import AppPage from './Containers/AppPage';
import registerServiceWorker from './registerServiceWorker';

// eslint-disable-next-line
import stringUtil from './utils/string';

ReactDOM.render(<AppPage />, document.getElementById('root'));

registerServiceWorker();
