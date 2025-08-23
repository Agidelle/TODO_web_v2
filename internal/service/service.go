package service

import (
	"context"
	"errors"
	"github.com/agidelle/merch-shop/internal/domain"
	"strconv"
	"strings"
	"time"
)

const limitSearch int = 25
const dateForm string = "20060102"

type TaskService struct {
	repo domain.TaskRepository
}

func NewService(repo domain.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) CloseDB() {
	s.repo.CloseDB()
}

func (s *TaskService) FindAll(ctx context.Context, filter *domain.Filter) ([]*domain.Task, *domain.CustomError) {
	switch {
	case filter.ID != nil:
		res, err := s.repo.FindTask(ctx, filter)
		if err != nil {
			return nil, domain.NewCustomError(0, domain.ErrInternalServer, err)
		}
		if len(res) == 0 {
			return nil, domain.NewCustomError(0, domain.ErrID, nil)
		}
		return res, nil
	case filter.SearchTerm != "":
		filter.Limit = limitSearch
		if date, err := time.Parse("02.01.2006", filter.SearchTerm); err == nil {
			filter.Date = date.Format(dateForm)
			filter.SearchTerm = ""
			tasks, err := s.repo.FindTask(ctx, filter)
			if err != nil {
				return nil, domain.NewCustomError(0, domain.ErrInternalServer, err)
			}
			return tasks, nil
		}

		tasks, err := s.repo.FindTask(ctx, filter)
		if err != nil {
			return nil, domain.NewCustomError(0, domain.ErrInternalServer, err)
		}
		return tasks, nil
	default:
		filter.Limit = limitSearch
		res, err := s.repo.FindTask(ctx, filter)
		if err != nil {
			return []*domain.Task{}, domain.NewCustomError(0, domain.ErrInternalServer, err)
		}
		return res, nil
	}
}

func (s *TaskService) Create(ctx context.Context, task *domain.Task) (int64, *domain.CustomError) {
	now := time.Now()
	nowF := now.Format(dateForm)

	//Проверки и исправления запроса
	if task.Title == "" {
		return 0, domain.NewCustomError(0, domain.ErrBadTitle, nil)
	}
	if task.Date == "" {
		task.Date = time.Now().Format(dateForm) //если дата пустая, присваиваем текущую
	}
	date, err := time.Parse(dateForm, task.Date)
	if err != nil {
		return 0, domain.NewCustomError(0, domain.ErrDate, err)
	}
	if task.Repeat == "" && nowF > date.Format(dateForm) {
		task.Date = nowF
	}
	if nowF > task.Date {
		task.Date, err = s.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			return 0, domain.NewCustomError(0, domain.ErrInternalServer, err)
		}
	}

	//Создаем задачу в БД
	id, err := s.repo.CreateTask(ctx, task)
	if err != nil {
		return 0, domain.NewCustomError(0, domain.ErrInternalServer, err)
	}
	return id, nil
}

func (s *TaskService) Update(ctx context.Context, task *domain.Task) *domain.CustomError {
	now := time.Now()
	nowF := now.Format(dateForm)
	//Проверки и исправления запроса
	if task.Title == "" {
		return domain.NewCustomError(0, domain.ErrBadTitle, nil)
	}
	if task.Date == "" {
		task.Date = time.Now().Format(dateForm)
	}
	date, err := time.Parse(dateForm, task.Date)
	if err != nil {
		return domain.NewCustomError(0, domain.ErrDate, err)
	}
	if task.Repeat == "" && nowF > date.Format(dateForm) {
		task.Date = nowF
	}
	if nowF > task.Date {
		task.Date, err = s.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			return domain.NewCustomError(0, domain.ErrInternalServer, err)
		}
	}
	err = s.repo.UpdateTask(ctx, task)
	if err != nil {
		return domain.NewCustomError(0, domain.ErrInternalServer, err)
	}
	return nil
}

func (s *TaskService) Done(ctx context.Context, filter *domain.Filter) *domain.CustomError {
	now := time.Now()
	task, err := s.repo.FindTask(ctx, filter)
	if err != nil {
		return domain.NewCustomError(0, domain.ErrID, err)
	}
	if len(task) == 0 {
		return domain.NewCustomError(0, domain.ErrID, nil)
	}
	rDay, err := s.NextDate(now, task[0].Date, task[0].Repeat)
	if err != nil {
		return domain.NewCustomError(0, domain.ErrInternalServer, err)
	}
	if rDay == "delete" {
		err = s.repo.DeleteTask(ctx, filter.ID)
		if err != nil {
			return domain.NewCustomError(0, domain.ErrInternalServer, err)
		}
		return nil
	}
	task[0].ID = strconv.Itoa(*filter.ID)
	task[0].Date = rDay
	err = s.repo.UpdateTask(ctx, task[0])
	if err != nil {
		return domain.NewCustomError(0, domain.ErrInternalServer, err)
	}
	return nil
}

func (s *TaskService) Delete(ctx context.Context, id int) *domain.CustomError {
	err := s.repo.DeleteTask(ctx, &id)
	if err != nil {
		return domain.NewCustomError(0, domain.ErrInternalServer, err)
	}
	return nil
}

func (s *TaskService) NextDate(now time.Time, dstart string, repeat string) (string, error) {
	var res time.Time
	pDate, err := time.Parse(dateForm, dstart)
	if err != nil {
		return "", errors.New("неправильный формат даты")
	}
	switch {
	case repeat == "":
		return "delete", nil
	case strings.HasPrefix(repeat, "d "):
		daysStr := strings.TrimPrefix(repeat, "d ")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days <= 0 || days > 400 {
			return "", errors.New("не более 400 дней")
		}
		if now.After(pDate) {
			diff := now.Sub(pDate)
			totalDays := int(diff.Hours() / 24)
			steps := totalDays/days + 1
			res = pDate.AddDate(0, 0, steps*days)
			return res.Format(dateForm), nil
		} else {
			pDate = pDate.AddDate(0, 0, days)
			return pDate.Format(dateForm), nil
		}
	case repeat == "y":
		if pDate.After(now) {
			pDate = pDate.AddDate(1, 0, 0)
			return pDate.Format(dateForm), nil
		} else {
			yearsDiff := now.Year() - pDate.Year()
			pDate = pDate.AddDate(yearsDiff, 0, 0)
			if !pDate.After(now) {
				pDate = pDate.AddDate(1, 0, 0)
			}
			return pDate.Format(dateForm), nil
		}
	case strings.HasPrefix(repeat, "w "):
		daysStr := strings.TrimPrefix(repeat, "w ")
		dayNumbers := strings.Split(daysStr, ",")
		var targetDays []int
		for _, day := range dayNumbers {
			dayNum, err := strconv.Atoi(day)
			if err != nil || dayNum < 1 || dayNum > 7 {
				return "", errors.New("неверный формат дней недели")
			}
			targetDays = append(targetDays, dayNum)
		}
		//Ищем ближайший день недели
		res, err = findNextWeekday(now, targetDays)
		if err != nil {
			return "", err
		}
		return res.Format(dateForm), nil
	case strings.HasPrefix(repeat, "m "):
		repeat = strings.TrimPrefix(repeat, "m ")
		parts := strings.Split(repeat, " ")
		if len(parts) == 0 || len(parts) > 2 {
			return "", errors.New("неверный формат правила повторения")
		}

		// Дни
		dayStrs := strings.Split(parts[0], ",")
		var targetDays []int
		for _, dayStr := range dayStrs {
			day, err := strconv.Atoi(dayStr)
			if err != nil || (day < -2 || day == 0 || day > 31) {
				return "", errors.New("неверный формат дней месяца")
			}
			targetDays = append(targetDays, day)
		}

		// Месяцы
		var targetMonths []int
		if len(parts) == 2 {
			monthStrs := strings.Split(parts[1], ",")
			for _, monthStr := range monthStrs {
				month, err := strconv.Atoi(monthStr)
				if err != nil || month < 1 || month > 12 {
					return "", errors.New("неверный формат месяцев")
				}
				targetMonths = append(targetMonths, month)
			}
		}

		res, err = findNextMonthDay(pDate, targetDays, targetMonths, now)
		if err != nil {
			return "", err
		}
		return res.Format(dateForm), nil
	default:
		return "", errors.New("неверный формат правила повторения")
	}
}

func findNextWeekday(now time.Time, targetDays []int) (time.Time, error) {
	currentDay := int(now.Weekday())
	if currentDay == 0 {
		currentDay = 7 // Преобразуем Sunday в 7 для удобства
	}
	//Поскольку в таком варианте, подходящий день может выпасть на сегодняшнее число,
	//то проверяем сразу и следующую неделю
	for i := 0; i < 14; i++ {
		nextDay := (currentDay + i) % 7
		if nextDay == 0 {
			nextDay = 7 // Преобразуем Sunday обратно в 7
		}
		for _, targetDay := range targetDays {
			if nextDay == targetDay {
				candidate := now.AddDate(0, 0, i)
				if candidate.After(now) {
					return candidate, nil
				}
			}
		}
	}

	return time.Time{}, errors.New("не удалось найти подходящий день недели")
}

func findNextMonthDay(dstart time.Time, targetDays []int, targetMonths []int, now time.Time) (time.Time, error) {
	currentYear, _, _ := dstart.Date()
	var variants []time.Time

	// Проверяем текущий и следующий год
	for yearOffset := 0; yearOffset < 2; yearOffset++ {
		for month := 1; month <= 12; month++ {
			if len(targetMonths) > 0 && !contains(targetMonths, month) {
				continue
			}

			lastDay := time.Date(currentYear+yearOffset, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()

			for _, day := range targetDays {
				targetDay := day
				if day == -1 {
					targetDay = lastDay
				} else if day == -2 {
					targetDay = lastDay - 1
				}
				if targetDay < 1 || targetDay > lastDay {
					continue
				}
				date := time.Date(currentYear+yearOffset, time.Month(month), targetDay, 0, 0, 0, 0, time.UTC)
				if !date.Before(dstart) && date.After(now) {
					variants = append(variants, date)
				}
			}
		}
	}
	if len(variants) == 0 {
		return time.Time{}, errors.New("не удалось найти подходящую дату")
	}

	//Находим минимальную дату
	minDate := variants[0]
	for _, date := range variants {
		if date.Before(minDate) {
			minDate = date
		}
	}
	return minDate, nil
}

func contains(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
