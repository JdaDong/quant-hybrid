import java.nio.charset.StandardCharsets
import java.nio.file.{Files, Paths}
import scala.util.Try

object FactorJob {
  final case class PriceRow(date: String, symbol: String, close: Double)

  private def parse(path: String): Vector[PriceRow] = {
    Files.readAllLines(Paths.get(path), StandardCharsets.UTF_8).toArray.toVector
      .drop(1)
      .map(_.toString.trim)
      .filter(_.nonEmpty)
      .map { line =>
        line.split(",", -1) match {
          case Array(date, symbol, close) => PriceRow(date, symbol, close.toDouble)
          case _ => throw new IllegalArgumentException(s"invalid price row: $line")
        }
      }
  }

  private def average(values: Vector[Double]): Double = values.sum / values.size

  def main(args: Array[String]): Unit = {
    if (args.length != 2) {
      Console.err.println("usage: FactorJob <prices.csv> <signals.csv>")
      sys.exit(2)
    }
    val rows = parse(args(0))
    val output = new StringBuilder("date,symbol,signal,score\n")
    rows.indices.foreach { index =>
      val shortWindow = rows.slice(math.max(0, index - 4), index + 1).map(_.close)
      val longWindow = rows.slice(math.max(0, index - 9), index + 1).map(_.close)
      val shortAverage = average(shortWindow)
      val longAverage = average(longWindow)
      val score = (shortAverage - longAverage) / longAverage
      val signal = if (longWindow.size < 10) 0 else if (score > 0.0) 1 else -1
      val row = rows(index)
      output.append(f"${row.date},${row.symbol},$signal,$score%.8f\n")
    }
    Files.write(Paths.get(args(1)), output.toString.getBytes(StandardCharsets.UTF_8))
    val active = math.max(0, rows.size - 9)
    println(s"{\"language\":\"scala\",\"job\":\"factor\",\"rows\":${rows.size},\"signals\":$active,\"output\":\"${args(1)}\"}")
  }
}
