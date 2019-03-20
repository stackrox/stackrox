import React from 'react';

import ApplyModification from './Dialogue/ApplyModification';
import NotifyModification from './Dialogue/NotifyModification';

export default function Dialogue() {
    return (
        <div className="border-none">
            <ApplyModification />
            <NotifyModification />
        </div>
    );
}
