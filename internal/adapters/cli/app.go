package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"diplomaMeetHelper/internal/domain"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type UserService interface {
	EnsureUser(ctx context.Context, userID string) (*domain.User, bool, error)
	GetUser(ctx context.Context, userID string) (*domain.User, error)
}

// CLIApp инкапсулирует CLI-слой на базе Cobra.
// Содержит зависимости бизнес-логики и потоки вывода:
//   - userService: интерфейс бизнес-логики
//   - userID: значение флага --user-id, изолированное в рамках экземпляра (без глобальных переменных).
//   - outWriter/errWriter: абстракции io.Writer (stdout/stderr) для изоляции от ОС и прямого перехвата вывода в Unit-тестах.
type CLIApp struct {
	userService UserService
	logger      *zap.Logger
	rootCmd     *cobra.Command
	userID      string
	outWriter   io.Writer
	errWriter   io.Writer
}

func NewApp(userService UserService, logger *zap.Logger) *CLIApp {
	app := &CLIApp{
		userService: userService,
		logger:      logger,
		outWriter:   os.Stdout,
		errWriter:   os.Stderr,
	}

	app.initCommands()
	return app
}

func (a *CLIApp) SetOutput(out io.Writer, errOut io.Writer) {
	a.outWriter = out
	a.errWriter = errOut
	a.rootCmd.SetOut(out)
	a.rootCmd.SetErr(errOut)
}

func (a *CLIApp) initCommands() {
	a.rootCmd = &cobra.Command{
		Use:   "diplomaMeetHelper",
		Short: "Помощник для обработки и анализа встреч",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" || (cmd.Name() == "diplomaMeetHelper" && len(args) == 0) {
				return nil
			}

			if strings.TrimSpace(a.userID) == "" {
				return errors.New("ошибка: флаг --user-id обязателен для выполнения команды")
			}

			if cmd.Name() != "start" {
				_, _, err := a.userService.EnsureUser(cmd.Context(), a.userID)
				if err != nil {
					return fmt.Errorf("ошибка проверки/регистрации пользователя: %w", err)
				}
			}

			return nil
		},
		SilenceUsage:  true,
		//необходимо потому-что мы сами логгируем ошибки
		SilenceErrors: true,
	}

	a.rootCmd.PersistentFlags().StringVarP(&a.userID, "user-id", "u", "", "Уникальный строковый идентификатор пользователя")
	a.rootCmd.AddCommand(a.newStartCmd())
}

func (a *CLIApp) newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Регистрация",
		RunE: func(cmd *cobra.Command, args []string) error {
			user, created, err := a.userService.EnsureUser(cmd.Context(), a.userID)
			if err != nil {
				return err
			}

			if created {
				fmt.Fprintf(a.outWriter, "Пользователь %s успешно зарегистрирован\n", user.ID)
			} else {
				fmt.Fprintf(a.outWriter, "Пользователь %s уже зарегистрирован\n", user.ID)
			}

			return nil
		},
	}
}

func (a *CLIApp) Execute(ctx context.Context, args []string) error {
	a.rootCmd.SetArgs(args)
	return a.rootCmd.ExecuteContext(ctx)
}

func (a *CLIApp) RootCmd() *cobra.Command {
	return a.rootCmd
}
