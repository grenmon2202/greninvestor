import pandas as pd
import yaml

INPUT_FILE = 'ind_nifty500list.csv'
OUTPUT_FILE = 'nifty500_symbols.yaml'

df = pd.read_csv(INPUT_FILE)

for i, x in df.iterrows():
    df.at[i, 'Symbol'] = x['Symbol'] + '.NS'
    
yaml_data = {
    'symbols': []
}

for i, x in df.iterrows():
    yaml_data['symbols'].append({
        'name': x['Company Name'],
        'code': x['Symbol'],
        'exchange': 'NSE',
        'currency': 'INR'
    })

with open(OUTPUT_FILE, 'w') as f:
    yaml.dump(yaml_data, f)