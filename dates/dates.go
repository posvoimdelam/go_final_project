package dates

import (
	"fmt"
	"go_final_project/models"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {

	start, err := time.Parse(models.Layout, date)
	if err != nil {
		return "", fmt.Errorf("invalid date format:%v", err)
	}

	if repeat == "" {
		return "", fmt.Errorf("empty repeat")
	}

	fields := strings.Fields(repeat)

	switch fields[0] {
	case "d":
		if len(fields) != 2 {
			return "", fmt.Errorf("invalid day rule")
		}

		days, err := strconv.Atoi(fields[1])
		if err != nil || days <= 0 || days > 400 {
			return "", fmt.Errorf("invalid day interval: %v", err)
		}
		for i := 0; i < 1000; i++ {
			start = start.AddDate(0, 0, days)
			if start.After(now) {
				return start.Format(models.Layout), nil
			}
		}
		return "", fmt.Errorf("exceeded iteration limit while calculating the next date")

	case "y":
		if len(fields) != 1 {
			return "", fmt.Errorf("invalid annual interval: %v", fields)
		}
		for {
			start = start.AddDate(1, 0, 0)
			if start.After(now) {
				return start.Format(models.Layout), nil
			}
		}

	case "w":
		if len(fields) != 2 {
			return "", fmt.Errorf("ivalid week rule")
		}
		daysOfWeek := strings.Split(fields[1], ",")
		for _, i := range daysOfWeek {
			num, err := strconv.Atoi(i)
			if !isValidDayOfWeek(num) || err != nil {
				return "", fmt.Errorf("invalid day of week format: %v", i)
			}
		}
		next := now.AddDate(0, 0, daysToAdd(now, daysOfWeek))
		return next.Format(models.Layout), nil
	case "m":
		if len(fields) > 3 || len(fields) < 2 || fields[1] == "" {
			return "", fmt.Errorf("invalid monthly rule")
		}

		daysOfMonth := strings.Split(fields[1], ",")

		var days []int //слайс дней
		for _, day := range daysOfMonth {
			num, err := strconv.Atoi(day)
			if err != nil || num < -2 || num > 31 || num == 0 {
				return "", fmt.Errorf("invalid day of month format: %v", err)
			}
			days = append(days, num)
		}

		var months []int //слайс месяцев
		if len(fields) == 3 {
			monthParts := strings.Split(fields[2], ",")
			for _, month := range monthParts {
				num, err := strconv.Atoi(month)
				if err != nil || num > 12 || num < 1 {
					return "", fmt.Errorf("invalid month format: %v", err)
				}
				months = append(months, num)
			}
		}

		if len(months) > 0 { //если есть месяцы правилах
			var nextDates []time.Time
			for _, month := range months {
				for _, day := range days {
					if day == -1 || day == -2 {
						// Последний или предпоследний день месяца
						date := lastOrPenultimateDay(now.Year(), time.Month(month), day)
						if date.After(now) && date.After(start) {
							nextDates = append(nextDates, date)
						}
					} else {
						// Конкретный день месяца
						if daysInMonth(time.Date(now.Year(), time.Month(month), 1, 0, 0, 0, 0, time.UTC)) < day {
							date := time.Date(now.Year(), time.Month(month+1), day, 0, 0, 0, 0, time.UTC)
							if date.After(now) && date.After(start) {
								nextDates = append(nextDates, date)
							}
							continue
						}
						date := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, time.UTC)
						if date.After(now) && date.After(start) {
							nextDates = append(nextDates, date)
						}
					}
				}
			}

			// Если нет дат в текущем году, добавляем даты на следующий год
			if len(nextDates) == 0 {
				for _, month := range months {
					for _, day := range days {
						if day == -1 || day == -2 {
							date := lastOrPenultimateDay(now.Year()+1, time.Month(month), day)
							nextDates = append(nextDates, date)
						} else {
							minYear := math.Min(float64(now.Year()), float64(start.Year()))

							if daysInMonth(time.Date(now.Year()+1, time.Month(month), 1, 0, 0, 0, 0, time.UTC)) < day {

								date := time.Date(int(minYear)+1, time.Month(month+1), day, 0, 0, 0, 0, time.UTC)
								nextDates = append(nextDates, date)
								continue
							}
							date := time.Date(int(minYear)+1, time.Month(month), day, 0, 0, 0, 0, time.UTC)
							nextDates = append(nextDates, date)
						}
					}
				}
			}

			// Находим ближайшую дату
			sort.Slice(nextDates, func(i, j int) bool {
				return nextDates[i].Before(nextDates[j])
			})

			if len(nextDates) > 0 {
				return nextDates[0].Format(models.Layout), nil
			}
		} else { //если в правилах только дни
			closestToStart := ifOnlyDays(date, days)
			if closestToStart.IsZero() { //если все даты либо до date либо совпадают с date, либо в месяце date меньше дней чем в правилах то
				nextMonth := start.AddDate(0, 1, 0)                                                                                 //добавляется месяц и
				closestToStart = time.Date(nextMonth.Year(), nextMonth.Month(), minimalDate(days, nextMonth), 0, 0, 0, 0, time.UTC) //формируется минимальная дата в новом месяце, ведь она будет ближе всего к date
			}
			return closestAfterNow(days, now, closestToStart).Format(models.Layout), nil
		}

	default:

		return "", fmt.Errorf("unsupported repeat rule:%v", repeat)
	}
	return "", nil
}

func closestAfterNow(days []int, now time.Time, closestToStart time.Time) time.Time { // возвращает ближайшую к date дату после now

	if closestToStart.After(now) { //если date после now, то closestToStar уже будет после now
		return closestToStart
	}

	dates := generateDates(days, now)

	if closestToStart.Before(now) { //если date до now то ищем ближайшую дату после now в месяце now

		for _, date := range dates {
			if date.After(now) { //так как dates отсортирован, первая дата после now будет ближайшей к date
				return date
			}
		}
		now = now.AddDate(0, 1, 0)
		return time.Date(now.Year(), now.Month(), minimalDate(days, now), 0, 0, 0, 0, time.UTC)
	}

	return time.Time{}
}

func generateDates(days []int, now time.Time) []time.Time { //создает слайс дат, из дней в правилах и месяца, переданных в аргументах
	if len(days) == 0 {
		return nil
	}
	var dates []time.Time
	for _, day := range days {
		if day == -1 || day == -2 {
			dates = append(dates, lastOrPenultimateDay(now.Year(), now.Month(), day))
		} else {
			date := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, time.UTC)
			dates = append(dates, date)
		}
	}
	sort.Slice(dates, func(i, j int) bool { //сортируем даты по возрастанию
		return dates[i].Before(dates[j])
	})
	return dates
}

func minimalDate(days []int, nextMonth time.Time) int { //возвращает минимальное число из правил учитывая что -1 и -2 это последний и предпоследний дни месяца
	if len(days) == 0 {
		return 0
	}

	var positiveNums []int
	var negativeNums []int
	for _, num := range days {
		if num > 0 {
			positiveNums = append(positiveNums, num)
		} else if num < 0 {
			negativeNums = append(negativeNums, num)
		}
	}

	var min float64
	if len(positiveNums) == 0 {
		min = float64(negativeNums[0])
		for _, num := range negativeNums {
			min = math.Min(min, float64(num))
		}
		return lastOrPenultimateDay(nextMonth.Year(), nextMonth.Month(), int(min)).Day()
	} else {
		min = float64(positiveNums[0])
		for _, num := range positiveNums {
			min = math.Min(min, float64(num))
		}
	}

	return int(min)
}

func ifOnlyDays(date string, days []int) time.Time { //возвращает ближайшую дату из правил после date, но только в рамках месяца date
	start, _ := time.Parse(models.Layout, date)

	var dates []time.Time

	for _, num := range days {
		if num == -1 || num == -2 {
			lastOrPenul := lastOrPenultimateDay(start.Year(), start.Month(), num)
			if lastOrPenul.After(start) {
				dates = append(dates, lastOrPenul)
			}
		} else {
			if daysInMonth(start) < num { //если в месяце меньше дней чем в правиле то итерация пропускается
				continue
			}
			newDate := time.Date(start.Year(), start.Month(), num, 0, 0, 0, 0, time.UTC)
			if newDate.After(start) {
				dates = append(dates, newDate)
			}
		}
	}

	if len(dates) == 0 { //если нет дат в рамках месяца и после date, то возвращается нулевое значение
		return time.Time{}
	}

	min := dates[0]
	for _, i := range dates {
		if i.Before(min) {
			min = i
		}
	}
	return min

}

func daysInMonth(date time.Time) int { //возвращает количество дней в месяце
	nextMonth := date.AddDate(0, 1, 0)
	date = time.Date(date.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	daysInMonth := date.AddDate(0, 0, -1)
	return daysInMonth.Day()
}

func lastOrPenultimateDay(year int, month time.Month, num int) time.Time { //возвращает последний(-1) или предпоследний(-2) день месяца
	nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastDay := nextMonth.AddDate(0, 0, -1)
	switch num {
	case -1:
		return lastDay
	case -2:
		penultimateDay := lastDay.AddDate(0, 0, -1)
		return penultimateDay
	default:
		return time.Time{}
	}
}

func isValidDayOfWeek(day int) bool { //проверка дня недели
	allowedDays := map[int]bool{
		1: true,
		2: true,
		3: true,
		4: true,
		5: true,
		6: true,
		7: true,
	}
	return allowedDays[day]
}

func daysToAdd(now time.Time, daysOfWeek []string) int { //вычисляет количество дней которое нужно добавить чтобы получить следующую дату

	todayNumber := int(now.Weekday())
	if todayNumber == 0 {
		todayNumber = 7
	}

	var moreSlice []int            //слайс дней после todayNumber
	var lessSlice []int            //слайс дней до todayNumber
	for _, i := range daysOfWeek { //сортируем дни из правил по слайсам
		num, _ := strconv.Atoi(i)
		if todayNumber > num {
			lessSlice = append(lessSlice, num)
		} else if todayNumber < num {
			moreSlice = append(moreSlice, num)
		}
	}

	var end float64
	if len(moreSlice) == 0 { //если нет дней после todayNumber то выбирается минимальный день до него и записывается в end
		end = float64(lessSlice[0])
		for _, i := range lessSlice {
			end = math.Min(end, float64(i))
		}
	} else {
		end = float64(moreSlice[0]) //в остальных случаях выбирается минимальный день после todayNumber и записывается в end
		for _, i := range moreSlice {
			end = math.Min(end, float64(i))
		}
	}
	var days int

	if todayNumber > int(end) {
		days = (int(end) + 7) - todayNumber //ищет разницу в днях между днем в текущей неделе(todayNumber) и днем на следующей неделе(end)
	} else {
		days = int(end) - todayNumber //разница в днях между текущим днем(todayNumber) и следующим днем из правил(end) в рамках одной недели
	}
	return days // возвращается количество дней которое нужно добавить
}
