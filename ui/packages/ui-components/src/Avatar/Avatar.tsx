import React, { ReactElement } from 'react';
import { find as createInitials } from 'initials';

/**
 * User avatar showing either provided image or person's initials.
 */
function Avatar({ imageSrc, name, extraClassName = '' }: AvatarProps): ReactElement {
    const finalClassName = `border border-base-400 rounded-full ${extraClassName}`;

    const initials = name ? createInitials(name) : '--';

    return imageSrc ? (
        <img src={imageSrc} alt={initials} className={finalClassName} />
    ) : (
        <span className={`text-xl ${finalClassName}`}>{initials}</span>
    );
}

export type AvatarProps = {
    /* URL to the avatar image */
    imageSrc?: string;
    /* person's full name to use for showing initials when image isn't available */
    name?: string;
    /* additional CSS classes for the top DOM element */
    extraClassName?: string;
};

export default Avatar;
