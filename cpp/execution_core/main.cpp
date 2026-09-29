#include <chrono>
#include <cstdlib>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <string>
#include <unordered_map>

namespace {

struct Order {
    std::string order_id;
    std::string symbol;
    std::string side;
    long quantity = 0;
    double limit_price = 0.0;
};

std::string stringField(const std::string& line, const std::string& key) {
    const std::string marker = "\"" + key + "\"";
    const auto keyPos = line.find(marker);
    if (keyPos == std::string::npos) return {};
    const auto colon = line.find(':', keyPos + marker.size());
    if (colon == std::string::npos) return {};
    const auto first = line.find('"', colon + 1);
    if (first == std::string::npos) return {};
    const auto last = line.find('"', first + 1);
    if (last == std::string::npos) return {};
    return line.substr(first + 1, last - first - 1);
}

double numberField(const std::string& line, const std::string& key, double fallback) {
    const std::string marker = "\"" + key + "\"";
    const auto keyPos = line.find(marker);
    if (keyPos == std::string::npos) return fallback;
    const auto colon = line.find(':', keyPos + marker.size());
    if (colon == std::string::npos) return fallback;
    const auto begin = line.find_first_of("-0123456789", colon + 1);
    if (begin == std::string::npos) return fallback;
    try {
        return std::stod(line.substr(begin));
    } catch (...) {
        return fallback;
    }
}

long long nowNs() {
    return std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
}

void emitRejected(const std::string& orderId, const std::string& reason) {
    std::cout << "{\"type\":\"reject\",\"order_id\":\"" << orderId
              << "\",\"reason\":\"" << reason << "\",\"ts_ns\":" << nowNs() << "}" << std::endl;
}

}  // namespace

int main() {
    std::ios::sync_with_stdio(false);
    std::cin.tie(nullptr);

    std::unordered_map<std::string, long> positions;
    double cash = 1'000'000.0;
    std::string line;

    while (std::getline(std::cin, line)) {
        if (line.empty()) continue;
        const auto type = stringField(line, "type");
        if (type == "query") {
            std::cout << "{\"type\":\"snapshot\",\"cash\":" << std::fixed << std::setprecision(2) << cash
                      << ",\"positions\":{";
            bool first = true;
            for (const auto& [symbol, quantity] : positions) {
                if (!first) std::cout << ',';
                first = false;
                std::cout << "\"" << symbol << "\":" << quantity;
            }
            std::cout << "},\"ts_ns\":" << nowNs() << "}" << std::endl;
            continue;
        }
        if (type != "order") {
            emitRejected({}, "unknown_type");
            continue;
        }

        Order order{
            stringField(line, "order_id"),
            stringField(line, "symbol"),
            stringField(line, "side"),
            static_cast<long>(numberField(line, "quantity", 0)),
            numberField(line, "limit_price", 0.0)
        };
        if (order.order_id.empty() || order.symbol.empty() || order.quantity <= 0 || order.limit_price <= 0.0) {
            emitRejected(order.order_id, "invalid_order");
            continue;
        }
        if (order.side != "BUY" && order.side != "SELL") {
            emitRejected(order.order_id, "invalid_side");
            continue;
        }

        std::cout << "{\"type\":\"accepted\",\"order_id\":\"" << order.order_id
                  << "\",\"ts_ns\":" << nowNs() << "}" << std::endl;

        const long signedQuantity = order.side == "BUY" ? order.quantity : -order.quantity;
        positions[order.symbol] += signedQuantity;
        cash += order.side == "BUY" ? -order.quantity * order.limit_price : order.quantity * order.limit_price;

        std::cout << "{\"type\":\"fill\",\"order_id\":\"" << order.order_id
                  << "\",\"symbol\":\"" << order.symbol
                  << "\",\"side\":\"" << order.side
                  << "\",\"filled_quantity\":" << order.quantity
                  << ",\"fill_price\":" << std::fixed << std::setprecision(2) << order.limit_price
                  << ",\"position\":" << positions[order.symbol]
                  << ",\"cash\":" << cash
                  << ",\"ts_ns\":" << nowNs() << "}" << std::endl;
    }
    return EXIT_SUCCESS;
}
