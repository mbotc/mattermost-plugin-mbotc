// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import React from 'react';

import {Svgs} from '../constants';

export default class Icon extends React.PureComponent {
    render() {
        return (
            <span
                className='d-flex align-items-center overflow--ellipsis icon'
                aria-hidden='true'
                dangerouslySetInnerHTML={{__html: Svgs.MBOTC_ICON}}
            />
        );
    }
}
