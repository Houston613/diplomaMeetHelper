package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type UserService interface {
	EnsureUser(ctx context.Context, userID string) (*domain.User, bool, error)
	GetUser(ctx context.Context, userID string) (*domain.User, error)
}

type MeetingService interface {
	LoadMeeting(ctx context.Context, userID string, filePath string) (uuid.UUID, error)
	GetMeetingStatus(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.JobStatusInfo, error)
	ListMeetings(ctx context.Context, userID string) ([]domain.MeetingListItem, error)
	GetMeetingDetails(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.MeetingDetails, error)
}

type SearchService interface {
	Search(ctx context.Context, userID string, query string) ([]domain.SearchResult, error)
}

type ChatService interface {
	Ask(ctx context.Context, userID string, meetingID uuid.UUID, question string) (string, error)
}

// Handler инкапсулирует CLI-слой на базе Cobra.
// Содержит зависимости бизнес-логики и потоки вывода:
//   - userService / meetingService / searchService / chatService: интерфейсы бизнес-логики (Dependency Inversion).
//   - userID: значение флага --user-id, изолированное в рамках экземпляра (без глобальных переменных).
//   - outWriter/errWriter: абстракции io.Writer (stdout/stderr) для изоляции от ОС и прямого перехвата вывода в Unit-тестах.
type Handler struct {
	userService    UserService
	meetingService MeetingService
	searchService  SearchService
	chatService    ChatService
	logger         *zap.Logger
	rootCmd        *cobra.Command
	userID         string
	meetingFlag    string
	outWriter      io.Writer
	errWriter      io.Writer
}

func NewHandler(
	userService UserService,
	meetingService MeetingService,
	searchService SearchService,
	chatService ChatService,
	logger *zap.Logger,
) *Handler {
	h := &Handler{
		userService:    userService,
		meetingService: meetingService,
		searchService:  searchService,
		chatService:    chatService,
		logger:         logger,
		outWriter:      os.Stdout,
		errWriter:      os.Stderr,
	}

	h.initCommands()
	return h
}

func (h *Handler) SetOutput(out io.Writer, errOut io.Writer) {
	h.outWriter = out
	h.errWriter = errOut
	h.rootCmd.SetOut(out)
	h.rootCmd.SetErr(errOut)
}

func (h *Handler) initCommands() {
	h.rootCmd = &cobra.Command{
		Use:   "diplomaMeetHelper",
		Short: "Помощник для обработки и анализа встреч",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" || (cmd.Name() == "diplomaMeetHelper" && len(args) == 0) {
				return nil
			}

			if strings.TrimSpace(h.userID) == "" {
				return errors.New("ошибка: флаг --user-id обязателен для выполнения команды")
			}

			if cmd.Name() != "start" {
				_, _, err := h.userService.EnsureUser(cmd.Context(), h.userID)
				if err != nil {
					return fmt.Errorf("ошибка проверки/регистрации пользователя: %w", err)
				}
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	h.rootCmd.PersistentFlags().StringVarP(&h.userID, "user-id", "u", "", "Уникальный строковый идентификатор пользователя")

	h.rootCmd.AddCommand(
		h.newStartCmd(),
		h.newLoadCmd(),
		h.newStatusCmd(),
		h.newListCmd(),
		h.newGetCmd(),
		h.newFindCmd(),
		h.newChatCmd(),
	)
}

func (h *Handler) newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Регистрация или приветствие пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			user, created, err := h.userService.EnsureUser(cmd.Context(), h.userID)
			if err != nil {
				return err
			}

			if created {
				fmt.Fprintf(h.outWriter, "Пользователь %s успешно зарегистрирован\n", user.ID)
			} else {
				fmt.Fprintf(h.outWriter, "Пользователь %s уже зарегистрирован\n", user.ID)
			}

			return nil
		},
	}
}

func (h *Handler) newLoadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load <file_path>",
		Short: "Загрузка аудиозаписи или текстового файла встречи на обработку",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			meetingID, err := h.meetingService.LoadMeeting(cmd.Context(), h.userID, filePath)
			if err != nil {
				return err
			}

			fmt.Fprintf(h.outWriter, "Файл %s принят в обработку. ID встречи: %s\n", filePath, meetingID.String())
			return nil
		},
	}
}

func (h *Handler) newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <meeting_id>",
		Short: "Получение статуса обработки встречи",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			meetingID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("некорректный формат meeting_id: %w", err)
			}

			statusInfo, err := h.meetingService.GetMeetingStatus(cmd.Context(), h.userID, meetingID)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(h.outWriter, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID встречи:\t%s\n", statusInfo.MeetingID)
			fmt.Fprintf(w, "ID задачи:\t%s\n", statusInfo.JobID)
			fmt.Fprintf(w, "Статус:\t%s\n", statusInfo.Status)
			fmt.Fprintf(w, "Количество попыток:\t%d\n", statusInfo.RetryCount)
			if statusInfo.ErrorMessage != "" {
				fmt.Fprintf(w, "Ошибка:\t%s\n", statusInfo.ErrorMessage)
			}
			fmt.Fprintf(w, "Дата создания:\t%s\n", statusInfo.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(w, "Дата обновления:\t%s\n", statusInfo.UpdatedAt.Format("2006-01-02 15:04:05"))
			w.Flush()

			return nil
		},
	}
}

func (h *Handler) newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Вывод списка сохраненных встреч пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := h.meetingService.ListMeetings(cmd.Context(), h.userID)
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Fprintln(h.outWriter, "Встречи не найдены.")
				return nil
			}

			w := tabwriter.NewWriter(h.outWriter, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tИмя файла\tСтатус\tДата создания\tДата обновления")
			for _, it := range items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					it.ID.String(),
					it.Filename,
					it.Status,
					it.CreatedAt.Format("2006-01-02 15:04:05"),
					it.UpdatedAt.Format("2006-01-02 15:04:05"),
				)
			}
			w.Flush()

			return nil
		},
	}
}

func (h *Handler) newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <meeting_id>",
		Short: "Получение полной информации, расшифровки и выжимки встречи",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			meetingID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("некорректный формат meeting_id: %w", err)
			}

			details, err := h.meetingService.GetMeetingDetails(cmd.Context(), h.userID, meetingID)
			if err != nil {
				return err
			}

			fmt.Fprintf(h.outWriter, "Встреча: %s (%s)\n", details.Meeting.Filename, details.Meeting.ID)
			fmt.Fprintf(h.outWriter, "Статус:  %s\n", details.Meeting.Status)
			fmt.Fprintf(h.outWriter, "Дата:    %s\n\n", details.Meeting.CreatedAt.Format(time.RFC3339))

			fmt.Fprintln(h.outWriter, "=== ТРАНСКРИПЦИЯ ===")
			if details.Transcript != nil {
				fmt.Fprintln(h.outWriter, details.Transcript.Content)
			}

			fmt.Fprintln(h.outWriter, "\n=== ВЫЖИМКА (SUMMARY) ===")
			if details.Summary != nil {
				fmt.Fprintln(h.outWriter, details.Summary.Content)
			}

			return nil
		},
	}
}

func (h *Handler) newFindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find <query>",
		Short: "Полнотекстовый поиск по расшифровкам и выжимкам встреч",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			results, err := h.searchService.Search(cmd.Context(), h.userID, query)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Fprintln(h.outWriter, "Ничего не найдено.")
				return nil
			}

			w := tabwriter.NewWriter(h.outWriter, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID встречи\tИмя файла\tИсточник\tДата\tФрагмент")
			for _, r := range results {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					r.MeetingID.String(),
					r.Filename,
					r.MatchSource,
					r.CreatedAt.Format("2006-01-02 15:04"),
					r.Snippet,
				)
			}
			w.Flush()

			return nil
		},
	}
}

func (h *Handler) newChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat [question...]",
		Short: "Интерактивный вопрос к LLM по контексту встречи",
		RunE: func(cmd *cobra.Command, args []string) error {
			var meetingID uuid.UUID
			var question string
			var err error

			if h.meetingFlag != "" {
				meetingID, err = uuid.Parse(h.meetingFlag)
				if err != nil {
					return fmt.Errorf("некорректный формат --meeting: %w", err)
				}
				question = strings.Join(args, " ")
			} else {
				if len(args) < 2 {
					return errors.New("использование: chat --meeting <meeting_id> <вопрос> или chat <meeting_id> <вопрос>")
				}
				meetingID, err = uuid.Parse(args[0])
				if err != nil {
					return fmt.Errorf("некорректный формат meeting_id: %w", err)
				}
				question = strings.Join(args[1:], " ")
			}

			answer, err := h.chatService.Ask(cmd.Context(), h.userID, meetingID, question)
			if err != nil {
				return err
			}

			fmt.Fprintf(h.outWriter, "Ответ: %s\n", answer)
			return nil
		},
	}

	cmd.Flags().StringVarP(&h.meetingFlag, "meeting", "m", "", "Идентификатор встречи (UUID)")
	return cmd
}

func (h *Handler) Execute(ctx context.Context, args []string) error {
	h.rootCmd.SetArgs(args)
	return h.rootCmd.ExecuteContext(ctx)
}

func (h *Handler) RootCmd() *cobra.Command {
	return h.rootCmd
}
