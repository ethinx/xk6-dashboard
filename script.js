import http from 'k6/http';
import { check } from 'k6';
import {
        randomItem
} from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
        summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'p(99.9)', 'p(99.99)', 'count'],
        noConnectionReuse: false,
        insecureSkipTLSVerify: true,
};

const targets = [
        {
                name: 'hello',
                url: 'http://192.168.66.209:8000/hi',
                headers: {},
                checker: {
                        'hello check': (r) => r.body.includes('Hi, there'),
                },
        },
        // {
        //         name: '404',
        //         url: 'http://192.168.66.209:8000/asdf',
        //         headers: {},
        //         checker: {
        //                 'hello check': (r) => r.body.includes('Hi, there'),
        //         },
        // },
        // {
        //       name: 'home',
        //       url: 'http://192.168.66.209:8000/',
        //       headers: {},
        //       checker: {
        //               'home check': (r) => r.body.includes('html'),
        //       },
        // },
        //{
        //      name: 'api',
        //      url: 'http://192.168.66.209:8000/api',
        //      headers: {},
        //      checker: {
        //              'api check': (r) => r.body.includes('Hi, there'),
        //      },
        //},
        //{
        //      name: 'private',
        //      url: 'http://192.168.66.209:8000/api/private',
        //      headers: {
        //              Authorization: 'Bearer ' + 'eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InNhbXBsZS1rZXktcnNhIn0.e30.gGCNd_UnnXsXE9QtJjxIG8vK-jpWDFqcHI2mqyydVnT2tckl6bpLDTc3zRZxagNCyk5myds9njhW41AAnQQ2XSxjFm4rgoHAOwKBKT_kyhYIvYYd_n3yVg9RwJytn1dXEC9OOmfFAbZCi6z7rwkCzEnu-lJQR9wxjw2RsFXCQAX6MkwEd7XTqTiIJ9jtt64jzCgKg-MshnVRGNOvGG7yivre0Nex4qiR00k--uF4LgioLgQK7CrohUnIXivQBY-bM9F2DRJpRX8Dj_q565wTF8B310vpUcmB5236Xuxinsh6-l7T_eD7Hnbd5S9M9sT-xNe0BdQdi6-hWsWDaP9pGQ',
        //      },
        //      checker: {
        //              'private check': (r) => r.body.includes('Staff only'),
        //      },
        //},
];

export default function () {
        let target = randomItem(targets);

        let resp = http.get(target.url, {
                headers: target.headers,
        });

        check(resp, target.checker);
}

