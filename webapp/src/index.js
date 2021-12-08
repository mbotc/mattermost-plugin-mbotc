// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.
/* eslint-disable */
import React from 'react';
import axios from 'axios';

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {id as pluginId} from './manifest';

const Icon = () => <svg version="1.1" id="Layer_1" xmlns="http://www.w3.org/2000/svg" xmlnsXlink="http://www.w3.org/1999/xlink" x="0px" y="0px" width="20px" height="20px" viewBox="0 0 20 20" enable-background="new 0 0 20 20" xmlSpace="preserve">  <image id="image0" width="20" height="20" x="0" y="0" href="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABQAAAAUCAYAAACNiR0NAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAABmJLR0QA/wD/AP+gvaeTAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAB3RJTUUH5QsEBhEAJbvZdQAAA/pJREFUOMttlF+IVVUUxn9r7X3OuefOzL1jJTM2ITY+pFRkiDEPRkH4omIipQhhFIbUg4S+RAQJEfgQ1KtvPTSp4ETPDf3R8kVQ8SFMS6gckMz541znzrn3nL1XD/dqFn6w2CwWfHzf2mstuA9ueDWI7kbdJ8AID8Yq4Hkge1DRA2RZxuDgEHmzzkzLbRWf7bXOnamR0VV/vbJ7L8MrVnDnTovz5y/y05nv94r6d60qtmRZdiWEQIyRGCMAAohzbnxoqLEBYXSpXeyrJN3kQvuL4eHh3/O8PqCqGmM055y/cXP2pbKK46mUnxn82imKNtAFrgMXBVi1dfvLU6+/+dZEmqZy4vgkJ788zoF33mbHzl3UajmqiplRVSWHDx1idnaWI0c+JMtqtFqLsSxLzv74w81TJ4/v98Cap5/ZsG7bjm0iArU8p9ls8sb+Azz51HpC+Lc/3W7Jxo3PUpUlO3e9Si1PMUO9hzWPj49+N/3NFg88ajHkf/5+HcNYu3Yt773/Ac57FuYXcc7hnENUUVU++vgoBqhTiqILZlTOkec5tTwf9qgf/3zqdPb1mWuoRLxTEq+kXkkTJfWOLHVkiSdNPGmaANApK7plABTUcevGH8zPzRdes8Zjy6NbpHzoCcwiFiNgWAwQAlYFaFdY7EeoAAN1qMsQ5xFxVHORorKWF1+vu3QQdZ7w9yVqS79h4lExBCOY9kZBwEmk5iNOQDAWO0Kx8gXcwEpscQDM2l5c0kQTQtVltV7j06P7qOUDOKeoOsqyi4jgnMP7hMHBAbz3eO85NfUVR078jBschdAxYtXyiGuIOmIoeXjFAJuem2BwqIGZoaoAxBgREUSE0P925xzr163Dx18wM6zqBIvlbUWkhigWSuqZQ1WpqooYI+fOnePq1auYGQsLC0xPT9PpdO5tRrPZINUSzLBYRqCtmCmAxZJa0rN2F5OTk1y4cAHnHDMzMxw7doyiKBARAGq1HC8Rw6BHWHgsBCxCjCROkL5NgD179jA2NkYIgbGxMQ4ePEi9XsfMensrINoTYDFEoOstlEVvJIzE9yyLCGbGxMQEZkaMkUajwebNm4kx0OfDADTtJbGKQMcTy9uEEpGEYnmJhblZslqOqLtnjXt3pEdjFhFgqdXCNLmfsOstdOesWiZpjnD6cpvtrx1mqJ7inP6HBJF+boAQTJlfMpbTcQTDLIQeYbl0IxYL+EfqVKt3cqUssLILndBrNverBBEF9Yh6pJ7hXF+hRQOCt9D9tjNz9sXQmhmxWA1g0WNREfGiaYLPEnGpF/UOcWJ3+2DRzIKVoYoWu7G8dfk2UN71MdiPOpACrv/m/6tlvWsAQOgf1gJoA/OIXPoHOiniGtPAqTcAAAAldEVYdGRhdGU6Y3JlYXRlADIwMjEtMTEtMDRUMDY6MTY6NTkrMDA6MDDMDc5GAAAAJXRFWHRkYXRlOm1vZGlmeQAyMDIxLTExLTA0VDA2OjE2OjU5KzAwOjAwvVB2+gAAAABJRU5ErkJggg==" /></svg>;
const clientURL = 'https://www.mbotc.com';

const getPluginServerRoute = (state) => {
    const config = getConfig(state);

    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;

        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath + '/plugins/' + pluginId;
};

class Plugin {
    sendRequest(postId, requestUrl) {
        axios.post(requestUrl, {post_id: postId})
        .then((res) => {
            console.log(res);
        })
        .catch((err) => {
            console.log(err);
        })
    }
    initialize(registry, store) {
        registry.registerChannelHeaderButtonAction(
            <Icon/>,
            () => {
                window.open(clientURL);
            },
            'MBotC',
        );
        registry.registerPostDropdownMenuAction(
            <div>
                {'Create MBotC Notice'}
            </div>,
            (postId) => {
                var requestUrl = getPluginServerRoute(store.getState()) + '/api/v1/create-notice-with-button';
                this.sendRequest(postId, requestUrl);
            },
        );
    }
}

window.registerPlugin(pluginId, new Plugin());
