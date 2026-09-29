import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

public final class Backtest {
    record Price(String date, String symbol, double close) {}

    private static List<String[]> readCsv(String path) throws IOException {
        List<String> lines = Files.readAllLines(Paths.get(path), StandardCharsets.UTF_8);
        List<String[]> rows = new ArrayList<>();
        for (int i = 1; i < lines.size(); i++) {
            String line = lines.get(i).trim();
            if (!line.isEmpty()) rows.add(line.split(",", -1));
        }
        return rows;
    }

    private static List<Price> readPrices(String path) throws IOException {
        List<Price> prices = new ArrayList<>();
        for (String[] row : readCsv(path)) {
            if (row.length != 3) throw new IllegalArgumentException("invalid price row");
            prices.add(new Price(row[0], row[1], Double.parseDouble(row[2])));
        }
        return prices;
    }

    private static Map<String, Integer> readSignals(String path) throws IOException {
        Map<String, Integer> signals = new HashMap<>();
        for (String[] row : readCsv(path)) {
            if (row.length != 4) throw new IllegalArgumentException("invalid signal row");
            signals.put(row[0], Integer.parseInt(row[2]));
        }
        return signals;
    }

    public static void main(String[] args) throws Exception {
        if (args.length != 2) {
            System.err.println("usage: Backtest <prices.csv> <signals.csv>");
            System.exit(2);
        }
        List<Price> prices = readPrices(args[0]);
        Map<String, Integer> signals = readSignals(args[1]);
        double equity = 1.0;
        double peak = equity;
        double maxDrawdown = 0.0;
        int trades = 0;
        int previousSignal = 0;

        for (int i = 0; i + 1 < prices.size(); i++) {
            Price current = prices.get(i);
            Price next = prices.get(i + 1);
            int signal = signals.getOrDefault(current.date(), 0);
            if (signal > 0 && previousSignal <= 0) trades++;
            previousSignal = signal;
            double dailyReturn = signal > 0 ? next.close() / current.close() - 1.0 : 0.0;
            equity *= 1.0 + dailyReturn;
            peak = Math.max(peak, equity);
            maxDrawdown = Math.min(maxDrawdown, equity / peak - 1.0);
        }

        double totalReturn = equity - 1.0;
        System.out.printf(Locale.ROOT,
                "{\"language\":\"java\",\"job\":\"backtest\",\"rows\":%d,\"total_return\":%.8f,\"max_drawdown\":%.8f,\"trades\":%d,\"lookahead\":false}%n",
                prices.size(), totalReturn, maxDrawdown, trades);
    }
}
