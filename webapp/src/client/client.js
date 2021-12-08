// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.
import {id} from '../manifest';

export default class Client {
    setServerRoute(url) {
        this.url = url + '/plugins/' + id;
    }
}
