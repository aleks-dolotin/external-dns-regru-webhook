import requests
import json
import time

BASE = 'http://127.0.0.1:8000/api/regru2'

def test_get_resource_records_ok():
    payload = {'input_format':'json','input_data': json.dumps({'domains':[{'dname':'example.com'}]})}
    r = requests.post(f'{BASE}/zone/get_resource_records', data=payload)
    assert r.status_code == 200
    j = r.json()
    assert j.get('result') == 'success'

def test_update_records_ok():
    payload = {'input_format':'json','input_data': json.dumps({'domains':[{'dname':'example.com','action_list':[{'action':'add_alias','subdomain':'www','ipaddr':'1.2.3.4'}]}]})}
    r = requests.post(f'{BASE}/zone/update_records', data=payload)
    assert r.status_code == 200
    j = r.json()
    assert j.get('result') == 'success'

def test_handle_429_and_retry():
    payload = {'input_format':'json','input_data': json.dumps({'simulate':'429'})}
    r = requests.post(f'{BASE}/zone/get_resource_records', data=payload)
    assert r.status_code == 429
    # if Retry-After header present, client should respect it; test just reads header
    ra = r.headers.get('Retry-After')
    assert ra is not None

if __name__ == '__main__':
    print('Run these tests with pytest after starting the mock server:')
    print('python tests/integration/mock-regru/server.py &')
    print('pytest -q tests/integration/mock-regru/test_mock_regu.py')

