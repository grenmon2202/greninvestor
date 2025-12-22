import pandas as pd
import yaml

INPUT_FILE = 'SnP_500.html'
OUTPUT_FILE = 'snp500_symbols.yaml'

df = pd.read_html(INPUT_FILE)[0]

yaml_data = {
    'symbols': []
}

for _, row in df.iterrows():
    yaml_data['symbols'].append({
        'name': str(row['Security']).strip(),
        'code': str(row['Symbol']).strip(),
        'exchange': 'NASDAQ',
        'currency': 'USD'
    })

with open(OUTPUT_FILE, 'w') as f:
    yaml.dump(yaml_data, f)
