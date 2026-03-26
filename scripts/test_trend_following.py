import unittest

from trend_following import TrendFollowingStrategy


class TrendFollowingSizingTests(unittest.TestCase):
    def setUp(self):
        self.strategy = TrendFollowingStrategy(
            {
                "SLOW_EMA_NUM_CANDLES": 50,
                "FAST_EMA_NUM_CANDLES": 20,
                "STOP_LOSS_PCT": 0.05,
                "CONFIDENCE_SCORE_MULTIPLIER": 100,
                "TRAILING_STOP_LOSS_PCT": 0.06,
                "MIN_TREND_SEPARATION": 0.01,
                "MAX_CAPITAL_PCT_PER_TRADE": 0.05,
            }
        )

    def test_high_confidence_buy_is_capped_by_max_capital(self):
        shares = self.strategy._compute_buy_shares(
            8.0,
            {"wallet": 1_000_000, "latest_price": 30.0},
        )

        self.assertEqual(shares, 1666)

    def test_low_confidence_buy_stays_below_cap(self):
        shares = self.strategy._compute_buy_shares(
            0.5,
            {"wallet": 100_000, "latest_price": 1_000.0},
        )

        self.assertEqual(shares, 5)

    def test_returns_zero_when_cap_cannot_buy_one_share(self):
        shares = self.strategy._compute_buy_shares(
            5.0,
            {"wallet": 10_000, "latest_price": 1_000.0},
        )

        self.assertEqual(shares, 0)


if __name__ == "__main__":
    unittest.main()
