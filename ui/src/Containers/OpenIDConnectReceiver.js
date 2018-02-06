
import React from 'react';

// TODO(cg): Replace this with a component that sets local storage for the access_token.
const OpenIDConnectReceiver = ({ location }) => {
    const params = location.hash.substr(1).split('&').map(param => (
        <div key={param.split('=')[0]}>
            <h4>{param.split('=')[0]}</h4>
            <p>{param.split('=')[1]}</p>
        </div>
    ));
    return (
        <div>
            <h3>URL Hash Parameters</h3>
            {params}
        </div>
    );
};

export default OpenIDConnectReceiver;
