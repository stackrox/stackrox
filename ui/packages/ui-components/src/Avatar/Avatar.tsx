import React, { ReactElement } from 'react';
import PropTypes, { InferProps } from 'prop-types';
import { find as createInitials } from 'initials';

/**
 * User avatar showing either provided image or person's initials.
 */
function Avatar({ imageSrc, name, extraClassName }: AvatarProps): ReactElement {
    const finalClassName = `border border-base-400 rounded-full ${extraClassName}`;

    const initials = name ? createInitials(name) : '--';

    return imageSrc ? (
        <img src={imageSrc} alt={initials} className={finalClassName} />
    ) : (
        <span className={`text-xl ${finalClassName}`}>{initials}</span>
    );
}

Avatar.propTypes = {
    /* URL to the avatar image */
    imageSrc: PropTypes.string,
    /* person's full name to use for showing initials when image isn't available */
    name: PropTypes.string,
    /* additional CSS classes for the top DOM element */
    extraClassName: PropTypes.string,
};

Avatar.defaultProps = {
    imageSrc: undefined,
    name: undefined,
    extraClassName: '',
};

export type AvatarProps = InferProps<typeof Avatar.propTypes>;
export default Avatar;
