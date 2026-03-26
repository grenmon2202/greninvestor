import argparse
import json
import yaml

class TrendFollowingStrategy:
    def __init__(self, config, debug=False, symbol=None):
        self.SLOW_EMA_NUM_CANDLES = config["SLOW_EMA_NUM_CANDLES"]
        self.FAST_EMA_NUM_CANDLES = config["FAST_EMA_NUM_CANDLES"]
        self.STOP_LOSS_PCT = config["STOP_LOSS_PCT"]
        self.CONFIDENCE_SCORE_MULTIPLIER = config["CONFIDENCE_SCORE_MULTIPLIER"]
        self.TRAILING_STOP_LOSS_PCT = config["TRAILING_STOP_LOSS_PCT"]
        self.MIN_TREND_SEPARATION = config["MIN_TREND_SEPARATION"]
        self.MAX_CAPITAL_PCT_PER_TRADE = config["MAX_CAPITAL_PCT_PER_TRADE"]
        
        self.symbol = symbol
        self.debug = debug
        
        if debug:
            import matplotlib.pyplot as plt
        
    def plot_ema_with_candles(self, candles, ema_s, ema_f, ema_s_prev, ema_f_prev):
        if not self.debug or not self.symbol:
            return
        
        # Extract timestamps and close prices
        timestamps = [candle['T'] for candle in candles]
        close_prices = [candle['C'] for candle in candles]
        
        # Create figure with single subplot
        fig, ax = plt.subplots(1, 1, figsize=(20, 12))
        
        # Plot OHLC as boxplots
        box_data = []
        for candle in candles:
            box_data.append([candle['O'], candle['H'], candle['L'], candle['C']])
        
        ax.boxplot(box_data, positions=range(len(candles)), widths=0.6, showfliers=False)
        
        # Overlay close price and EMAs
        ax.plot(range(len(timestamps)), close_prices, label='Close Price', color='blue', alpha=0.7, linewidth=3)
        ax.plot(range(len(timestamps) - len(ema_s), len(timestamps)), ema_s, color='red', linestyle='--', label=f'Slow EMA ({self.SLOW_EMA_NUM_CANDLES})', linewidth=3)
        ax.plot(range(len(timestamps) - len(ema_f), len(timestamps)), ema_f, color='green', linestyle='--', label=f'Fast EMA ({self.FAST_EMA_NUM_CANDLES})', linewidth=3)
        
        # Plot previous EMAs (offset by 1 position)
        ax.plot(range(len(timestamps) - len(ema_s_prev) - 1, len(timestamps) - 1), ema_s_prev, color='magenta', linestyle=':', alpha=0.5, label=f'Slow EMA Prev ({self.SLOW_EMA_NUM_CANDLES})', linewidth=2)
        ax.plot(range(len(timestamps) - len(ema_f_prev) - 1, len(timestamps) - 1), ema_f_prev, color='#045c5a', linestyle=':', alpha=0.5, label=f'Fast EMA Prev ({self.FAST_EMA_NUM_CANDLES})', linewidth=2)
        
        # Set x-axis to show timestamps
        ax.set_xticks(range(len(timestamps)))
        ax.set_xticklabels([t.split('+')[0] for t in timestamps], rotation=45, ha='right', fontsize=10)
        
        ax.set_xlabel('Timestamp', fontsize=14)
        ax.set_ylabel('Price', fontsize=14)
        ax.set_title(f'{self.symbol} - OHLC with EMAs', fontsize=16)
        ax.legend(fontsize=12)
        ax.grid(True, alpha=0.3)
        ax.tick_params(axis='both', which='major', labelsize=11)
        
        plt.tight_layout()
        filename = f"debug/{self.symbol}_trend_following.png"
        plt.savefig(filename, dpi=150, bbox_inches='tight')
        plt.close()
        
    def compute_ema(self, candles, num_candles):
        alpha = 2 / (num_candles + 1)
        
        compute_candles = candles[-num_candles:]
        
        EMA = []
        EMA.append(compute_candles[0]['C'])
        
        for i in range(1, num_candles):
            EMA.append(alpha * compute_candles[i]['C'] + (1 - alpha) * EMA[i - 1])
            
        return EMA[-1], EMA
    
    def _ema_crossed(self, data):
        ema_f, ema_f_debug = self.compute_ema(data, self.FAST_EMA_NUM_CANDLES)
        ema_s, ema_s_debug = self.compute_ema(data, self.SLOW_EMA_NUM_CANDLES)
        
        ema_f_prev, ema_f_prev_debug = self.compute_ema(data[:-1], self.FAST_EMA_NUM_CANDLES)
        ema_s_prev, ema_s_prev_debug = self.compute_ema(data[:-1], self.SLOW_EMA_NUM_CANDLES)
        
        if self.debug:
            self.plot_ema_with_candles(data, ema_s_debug, ema_f_debug, ema_s_prev_debug, ema_f_prev_debug)
        
        conf = abs((ema_f - ema_s) / data[-1]['C']) * self.CONFIDENCE_SCORE_MULTIPLIER
        
        if ema_f > ema_s and ema_f_prev <= ema_s_prev:
            return 1, conf
        elif ema_f < ema_s and ema_f_prev >= ema_s_prev:
            return -1, conf
        else:
            return 0, conf
        
    def _trigger_stop_loss(self, data, port_data):
        current_price = data[-1]['C']
        if current_price <= port_data['entry_point'] * (1 - self.STOP_LOSS_PCT):
            return True
        return False
    
    def _trigger_trailing_stop_loss(self, data, port_data):
        current_price = data[-1]['C']
        highest_price = max(candle['C'] for candle in data if candle['T'] >= port_data['T'])
        
        trailing_stop_price = highest_price * (1 - self.TRAILING_STOP_LOSS_PCT)
        
        if current_price <= trailing_stop_price:
            return True
        return False
    
    def _validate_data(self, data):
        if len(data) < max(self.SLOW_EMA_NUM_CANDLES, self.FAST_EMA_NUM_CANDLES):
            raise ValueError("Insufficient data length for EMA calculation")
        
        for candle in data:
            if 'C' not in candle or 'T' not in candle:
                raise ValueError("Candle data must contain 'C' (close price) and 'T' (timestamp)")

    def _compute_buy_shares(self, conf, buy_ctx):
        wallet = buy_ctx.get("wallet")
        latest_price = buy_ctx.get("latest_price")

        if wallet is None or latest_price is None:
            raise ValueError("Buy context must include 'wallet' and 'latest_price'")

        if wallet <= 0 or latest_price <= 0:
            return 0

        confidence_ratio = max(0.05, min(conf / 10.0, 0.25))
        target_capital = wallet * confidence_ratio
        max_capital = wallet * self.MAX_CAPITAL_PCT_PER_TRADE

        max_shares_by_cap = int(max_capital / latest_price)
        if max_shares_by_cap <= 0:
            return 0

        raw_shares = max(1, int(target_capital / latest_price))
        return min(raw_shares, max_shares_by_cap)

    def buy(self, data, buy_ctx):
        self._validate_data(data)
        ema_cross, conf = self._ema_crossed(data)
        if ema_cross == 1:
            shares = self._compute_buy_shares(conf, buy_ctx)
            return True, conf, shares

        return False, conf, 0
        
    def sell(self, data, port_data):
        self._validate_data(data)
        if self._trigger_stop_loss(data, port_data):
            return True, -1, 0
        
        if self._trigger_trailing_stop_loss(data, port_data):
            return True, -1, 0
        
        ema_cross, conf = self._ema_crossed(data)
        if ema_cross == -1:
            return True, conf, 0
        
        return False, conf, 0
        
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Trend Following Strategy")
    parser.add_argument(
        "--config", "-c",
        type=str,
        default="config/trend_following.yaml",
        help="Path to the configuration file",
    )
    parser.add_argument(
        "--mode", "-m",
        type=str,
        required=True,
        choices=["buy", "sell"],
        help="Mode to run the strategy: buy or sell",
    )
    parser.add_argument(
        "--input", "-i",
        type=str,
        required=True,
        help="Input json file with historical candle data"
    )
    parser.add_argument(
        "--port_data", "-p",
        type=str,
        required=False,
        help="Input json file with portfolio data or buy context"
    )
    parser.add_argument(
        "--debug", "-d",
        action="store_true",
        help="Enable debug mode"
    )
    parser.add_argument(
        "--symbol", "-s",
        type=str,
        required=False,
        help="Trading symbol (optional)"
    )
    args = parser.parse_args()
    
    if args.debug and not args.symbol:
        raise ValueError("Symbol must be provided in debug mode")

    with open(args.config, "r") as file:
        config = yaml.safe_load(file)
        
    strategy = TrendFollowingStrategy(config, debug=args.debug, symbol=args.symbol)
    
    data = json.loads(args.input)
    
    if args.mode == "buy":
        if args.port_data is None:
            raise ValueError("Buy context must be provided in buy mode")

        buy_ctx = json.loads(args.port_data)
        buy, conf, shares = strategy.buy(data, buy_ctx)
        print(json.dumps({"decision": buy, "confidence": conf, "shares": shares}))
    elif args.mode == "sell":
        if args.port_data is None:
            raise ValueError("Portfolio data file must be provided in sell mode")
        
        port_data = json.loads(args.port_data)
        sell, conf, shares = strategy.sell(data, port_data)
        print(json.dumps({"decision": sell, "confidence": conf, "shares": shares}))
