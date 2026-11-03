import { request } from '@/util/request'

export const getIP = (node_id = 0) => request(`/api/detect/ip?node_id=${node_id}`)
export const fetchIP = (source: string, node_id = 0) =>
    request('/api/detect/ip/fetch', {
        method: 'POST',
        body: JSON.stringify({ source, node_id }),
    })

export const getUnlock = (node_id = 0) => request(`/api/detect/unlock?node_id=${node_id}`)
export const fetchUnlock = (node_id = 0) =>
    request('/api/detect/unlock/fetch', {
        method: 'POST',
        body: JSON.stringify({node_id}),
    })

export const getBackRoute = () => request('/api/detect/back-route')
export const fetchBackRoute = () =>
    request('/api/detect/back-route/fetch', {
        method: 'POST',
    })
